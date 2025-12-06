package app

import (
	"bufio"
	"encoding/json"
	"log"
	"strings"

	nmea "github.com/adrianmo/go-nmea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	serial "github.com/jacobsa/go-serial/serial"

	"github.com/relabs-tech/inertial_computer/internal/gps"
)

// RunGPSProducer opens the GPS serial port, parses NMEA sentences, and
// publishes combined GPS fixes as JSON to MQTT topic "inertial/gps".
func RunGPSProducer() error {
	// ---- 1) Connect to MQTT broker ----
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-gps-producer")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Println("GPS producer connected to MQTT broker at tcp://localhost:1883")

	// ---- 2) Open GPS serial port ----
	// NOTE: adjust PortName to match your setup: /dev/serial0, /dev/ttyAMA0, /dev/ttyUSB0, etc.
	serialOpts := serial.OpenOptions{
		PortName:              "/dev/serial0",
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       1,
		ParityMode:            serial.PARITY_NONE,
		InterCharacterTimeout: 0,
	}

	port, err := serial.Open(serialOpts)
	if err != nil {
		return err
	}
	defer port.Close()
	log.Printf("GPS serial port opened on %s at %d baud", serialOpts.PortName, serialOpts.BaudRate)

	reader := bufio.NewReader(port)

	// We'll accumulate data mainly from RMC; you can extend to use GGA/GSA/etc.
	var current gps.Fix

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("GPS read error: %v", err)
			return err // or continue if you prefer to keep trying
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// NMEA sentences usually start with '$'
		if !strings.HasPrefix(line, "$") {
			continue
		}

		sentence, err := nmea.Parse(line)
		if err != nil {
			// noisy GPS or partial sentences; log at debug if too chatty
			// log.Printf("NMEA parse error: %v (line: %q)", err, line)
			continue
		}

		switch sentence.DataType() {
		case nmea.TypeRMC:
			m := sentence.(nmea.RMC)

			// Fill Fix from RMC data
			current.Time = m.Time.String()  // e.g. "12:34:56"
			current.Date = m.Date.String()  // library format, fine for now
			current.Latitude = m.Latitude   // decimal degrees
			current.Longitude = m.Longitude // decimal degrees
			current.SpeedKnots = m.Speed    // already in knots
			current.CourseDeg = m.Course    // in degrees
			current.Validity = string(m.Validity)

			// Publish each RMC as one GPS fix
			payload, err := json.Marshal(current)
			if err != nil {
				log.Printf("GPS JSON marshal error: %v", err)
				continue
			}

			token := client.Publish("inertial/gps", 0, true, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("GPS publish error: %v", token.Error())
				continue
			}

			log.Printf("published GPS fix: %+v", current)

		default:
			// ignore other sentence types for now (GGA, GSA, etc.)
		}
	}
}
