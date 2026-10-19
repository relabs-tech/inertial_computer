// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/gps"
)

// gpsdEnvelope is used to peek at the "class" field of each gpsd JSON
// report before decoding it into its specific type.
type gpsdEnvelope struct {
	Class string `json:"class"`
}

// gpsdTPV mirrors gpsd's "TPV" (time-position-velocity) report.
type gpsdTPV struct {
	Device string  `json:"device"`
	Mode   int     `json:"mode"` // 0/1=no fix, 2=2D, 3=3D
	Time   string  `json:"time"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Alt    float64 `json:"alt"`    // deprecated alias, older gpsd versions
	AltMSL float64 `json:"altMSL"` // height above mean sea level
	Speed  float64 `json:"speed"`  // m/s
	Track  float64 `json:"track"`  // degrees true
}

// gpsdSatellite mirrors one entry of gpsd's "SKY" satellites array.
type gpsdSatellite struct {
	Used bool `json:"used"`
}

// gpsdSKY mirrors gpsd's "SKY" (satellites/DOP) report.
type gpsdSKY struct {
	HDOP       float64         `json:"hdop"`
	PDOP       float64         `json:"pdop"`
	VDOP       float64         `json:"vdop"`
	Satellites []gpsdSatellite `json:"satellites"`
}

// gpsdPPS mirrors gpsd's "PPS" report: the PPS pulse timestamp (real_*) vs
// the system clock reading taken at that same edge (clock_*).
type gpsdPPS struct {
	Device    string `json:"device"`
	RealSec   int64  `json:"real_sec"`
	RealNsec  int64  `json:"real_nsec"`
	ClockSec  int64  `json:"clock_sec"`
	ClockNsec int64  `json:"clock_nsec"`
}

// RunGPSDProducer bridges gpsd's JSON stream to MQTT for the *system* GPS
// (the module on /dev/ttyAMA3 with PPS wired to GPIO6, dedicated to chrony
// time discipline). gpsd holds that device exclusively, so it cannot be
// read directly as raw NMEA the way gps_producer reads /dev/serial0 - this
// producer connects to gpsd's network JSON protocol instead.
func RunGPSDProducer() error {
	cfg := config.Get()

	// ---- 1) Connect to MQTT broker ----
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDGPSD)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("gpsd producer connected to MQTT broker at %s", cfg.MQTTBroker)

	// ---- 2) Connect to gpsd and start watching ----
	addr := net.JoinHostPort(cfg.GPSDHost, strconv.Itoa(cfg.GPSDPort))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to gpsd at %s: %w", addr, err)
	}
	defer conn.Close()
	log.Printf("gpsd producer connected to gpsd at %s", addr)

	watch := `?WATCH={"enable":true,"json":true}`
	if cfg.GPSDDevice != "" {
		watch = fmt.Sprintf(`?WATCH={"enable":true,"json":true,"device":%q}`, cfg.GPSDDevice)
	}
	if _, err := fmt.Fprintf(conn, "%s\n", watch); err != nil {
		return fmt.Errorf("send gpsd WATCH command: %w", err)
	}

	var current gps.SystemFix
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("gpsd producer: read error: %v", err)
			return err
		}

		var env gpsdEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}

		switch env.Class {
		case "TPV":
			var tpv gpsdTPV
			if err := json.Unmarshal(line, &tpv); err != nil {
				log.Printf("gpsd producer: TPV decode error: %v", err)
				continue
			}
			current.Device = tpv.Device
			current.Time = tpv.Time
			current.Mode = tpv.Mode
			switch tpv.Mode {
			case 3:
				current.FixType = "3D"
			case 2:
				current.FixType = "2D"
			default:
				current.FixType = "no fix"
			}
			current.Latitude = tpv.Lat
			current.Longitude = tpv.Lon
			if tpv.AltMSL != 0 {
				current.AltitudeM = tpv.AltMSL
			} else {
				current.AltitudeM = tpv.Alt
			}
			current.SpeedKnots = tpv.Speed * 1.9438445 // m/s -> knots
			current.CourseDeg = tpv.Track

		case "SKY":
			var sky gpsdSKY
			if err := json.Unmarshal(line, &sky); err != nil {
				log.Printf("gpsd producer: SKY decode error: %v", err)
				continue
			}
			current.HDOP = sky.HDOP
			current.PDOP = sky.PDOP
			current.VDOP = sky.VDOP
			current.SatellitesSeen = len(sky.Satellites)
			used := 0
			for _, s := range sky.Satellites {
				if s.Used {
					used++
				}
			}
			current.SatellitesUsed = used

		case "PPS":
			var pps gpsdPPS
			if err := json.Unmarshal(line, &pps); err != nil {
				log.Printf("gpsd producer: PPS decode error: %v", err)
				continue
			}
			current.HavePPS = true
			current.PPSDevice = pps.Device
			current.PPSOffsetSec = float64(pps.ClockSec-pps.RealSec) + float64(pps.ClockNsec-pps.RealNsec)/1e9

		default:
			continue
		}

		payload, err := json.Marshal(current)
		if err != nil {
			log.Printf("gpsd producer: JSON marshal error: %v", err)
			continue
		}
		token := client.Publish(cfg.TopicGPSSystem, 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("gpsd producer: publish error: %v", token.Error())
		}
	}
}
