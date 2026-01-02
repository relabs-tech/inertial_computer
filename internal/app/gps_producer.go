package app

import (
	"bufio"
	"encoding/json"
	"log"
	"strings"

	nmea "github.com/adrianmo/go-nmea"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	serial "github.com/jacobsa/go-serial/serial"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/gps"
)

// RunGPSProducer opens the GPS serial port, parses NMEA sentences, and
// publishes combined GPS fixes as JSON to MQTT.
func RunGPSProducer() error {
	cfg := config.Get()

	// ---- 1) Connect to MQTT broker ----
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDGPS)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
		return token.Error()
	}
	log.Printf("GPS producer connected to MQTT broker at %s", cfg.MQTTBroker)

	// ---- 2) Open GPS serial port ----
	serialOpts := serial.OpenOptions{
		PortName:              cfg.GPSSerialPort,
		BaudRate:              uint(cfg.GPSBaudRate),
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

	// Accumulate data from multiple NMEA sentence types.
	// Publish to separate topics for different data categories.
	var position gps.Position
	var velocity gps.Velocity
	var quality gps.Quality

	// For backwards compatibility, maintain full Fix
	var current gps.Fix
	lastPublishedFull := ""

	// GSV messages come in multiple parts - accumulate satellites across messages
	// Separate buffers for GPS (GPGSV) and GLONASS (GLGSV)
	var gpsSatelliteBuffer []gps.Satellite
	var glonassSatelliteBuffer []gps.Satellite

	// Helper to publish to a topic
	publishJSON := func(topic string, data interface{}) {
		payload, err := json.Marshal(data)
		if err != nil {
			log.Printf("JSON marshal error for %s: %v", topic, err)
			return
		}
		token := client.Publish(topic, 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Publish error to %s: %v", topic, token.Error())
		}
	}

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

		// Log all raw data received
		log.Printf("[GPS-RAW] %s", line)

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
			// RMC: Recommended Minimum - provides time, date, position, speed, course
			m := sentence.(nmea.RMC)

			// Update position
			position.Time = m.Time.String()
			position.Date = m.Date.String()
			position.Latitude = m.Latitude
			position.Longitude = m.Longitude
			position.Validity = string(m.Validity)

			// Update velocity
			velocity.SpeedKnots = m.Speed
			velocity.CourseDeg = m.Course

			// Update full fix
			current.Time = m.Time.String()
			current.Date = m.Date.String()
			current.Latitude = m.Latitude
			current.Longitude = m.Longitude
			current.SpeedKnots = m.Speed
			current.CourseDeg = m.Course
			current.Validity = string(m.Validity)

			// Publish position and velocity to separate topics
			publishJSON(cfg.TopicGPSPosition, position)
			publishJSON(cfg.TopicGPSVelocity, velocity)

			// Publish full fix to legacy topic (for backwards compatibility)
			payloadFull, err := json.Marshal(current)
			if err != nil {
				log.Printf("GPS JSON marshal error: %v", err)
				continue
			}

			payloadStr := string(payloadFull)
			if payloadStr != lastPublishedFull {
				publishJSON(cfg.TopicGPS, current)
				totalSats := len(current.GPSSatellitesInView) + len(current.GLONASSSatellitesInView)
				log.Printf("published GPS: lat=%.6f lon=%.6f alt=%.1fm sats=%d/%d fix=%s",
					current.Latitude, current.Longitude, current.Altitude,
					current.NumSatellites, totalSats, current.FixType)
				lastPublishedFull = payloadStr
			}

		case nmea.TypeGGA:
			// GGA: Global Positioning System Fix Data - provides altitude, fix quality, satellites
			m := sentence.(nmea.GGA)

			// Update position with altitude
			position.Altitude = m.Altitude

			// Update quality
			quality.NumSatellites = m.NumSatellites
			quality.HDOP = m.HDOP

			// Map fix quality to descriptive string
			switch m.FixQuality {
			case "0":
				quality.FixQuality = "invalid"
			case "1":
				quality.FixQuality = "GPS"
			case "2":
				quality.FixQuality = "DGPS"
			case "4":
				quality.FixQuality = "RTK fixed"
			case "5":
				quality.FixQuality = "RTK float"
			default:
				quality.FixQuality = m.FixQuality
			}

			// Update full fix
			current.Altitude = m.Altitude
			current.NumSatellites = m.NumSatellites
			current.HDOP = m.HDOP
			current.FixQuality = quality.FixQuality

			// Publish position and quality
			publishJSON(cfg.TopicGPSPosition, position)
			publishJSON(cfg.TopicGPSQuality, quality)

		case nmea.TypeGSA:
			// GSA: GPS DOP and Active Satellites - provides fix type and dilution of precision
			m := sentence.(nmea.GSA)

			// Map fix type to descriptive string
			switch m.FixType {
			case "1":
				quality.FixType = "no fix"
			case "2":
				quality.FixType = "2D"
			case "3":
				quality.FixType = "3D"
			default:
				quality.FixType = m.FixType
			}

			quality.PDOP = m.PDOP
			quality.HDOP = m.HDOP
			quality.VDOP = m.VDOP

			// Update full fix
			current.FixType = quality.FixType
			current.PDOP = m.PDOP
			current.HDOP = m.HDOP
			current.VDOP = m.VDOP

			// Publish quality
			publishJSON(cfg.TopicGPSQuality, quality)

		case nmea.TypeVTG:
			// VTG: Track Made Good and Ground Speed - provides speed in km/h
			m := sentence.(nmea.VTG)

			velocity.SpeedKmh = m.GroundSpeedKPH
			current.SpeedKmh = m.GroundSpeedKPH

			// Publish velocity
			publishJSON(cfg.TopicGPSVelocity, velocity)

		case nmea.TypeGSV:
			// GSV: GPS Satellites in View - provides satellite info with signal strength
			m := sentence.(nmea.GSV)

			// Determine constellation type from the raw sentence (GPGSV vs GLGSV)
			isGPS := strings.HasPrefix(line, "$GPGSV")
			isGLONASS := strings.HasPrefix(line, "$GLGSV")

			if !isGPS && !isGLONASS {
				// Skip other constellations for now (GAGSV, GBGSV, etc.)
				continue
			}

			// GSV messages can span multiple sentences (1 of 3, 2 of 3, etc.)
			// MessageNumber and TotalMessages tell us which part we're on

			// If this is the first message in the sequence, reset the buffer
			if m.MessageNumber == 1 {
				if isGPS {
					gpsSatelliteBuffer = make([]gps.Satellite, 0)
				} else if isGLONASS {
					glonassSatelliteBuffer = make([]gps.Satellite, 0)
				}
			}

			// Add satellites from this GSV message to the appropriate buffer
			for _, sv := range m.Info {
				sat := gps.Satellite{
					SVNumber:  sv.SVPRNNumber,
					Elevation: sv.Elevation,
					Azimuth:   sv.Azimuth,
					SNR:       sv.SNR,
				}
				if isGPS {
					gpsSatelliteBuffer = append(gpsSatelliteBuffer, sat)
				} else if isGLONASS {
					glonassSatelliteBuffer = append(glonassSatelliteBuffer, sat)
				}
			}

			// If this is the last message in the sequence, publish satellites
			if m.MessageNumber == m.TotalMessages {
				if isGPS {
					// Publish only GPS satellites (no GLONASS fields)
					gpsOnly := struct {
						Satellites []gps.Satellite `json:"satellites"`
						Count      int             `json:"count"`
					}{
						Satellites: gpsSatelliteBuffer,
						Count:      len(gpsSatelliteBuffer),
					}
					current.GPSSatellitesInView = gpsSatelliteBuffer

					publishJSON(cfg.TopicGPSSatellites, gpsOnly)
					log.Printf("[GPS-SAT] GPS satellites: %d visible", len(gpsSatelliteBuffer))
				} else if isGLONASS {
					// Publish only GLONASS satellites (no GPS fields)
					glonassOnly := struct {
						Satellites []gps.Satellite `json:"satellites"`
						Count      int             `json:"count"`
					}{
						Satellites: glonassSatelliteBuffer,
						Count:      len(glonassSatelliteBuffer),
					}
					current.GLONASSSatellitesInView = glonassSatelliteBuffer

					publishJSON(cfg.TopicGLONASSSatellites, glonassOnly)
					log.Printf("[GPS-SAT] GLONASS satellites: %d visible", len(glonassSatelliteBuffer))
				}
			}

		default:
			// Ignore other sentence types (GLL, etc.)
		}
	}
}
