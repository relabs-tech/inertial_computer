package app

import (
	"encoding/json"
	"log"
	"math"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/config"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

// magNorm computes the magnitude of the magnetic field vector.
// This is TEST/DEBUG code to validate magnetometer behavior end-to-end.
func magNorm(mx, my, mz int16) float64 {
	x := float64(mx)
	y := float64(my)
	z := float64(mz)
	return math.Sqrt(x*x + y*y + z*z)
}

func RunInertialProducer() error {
	log.Println("starting inertial-computer orientation/env producer")

	cfg := config.Get()

	// --- Initialize IMU manager (both left and right) ---
	imuManager := sensors.GetIMUManager()
	if err := imuManager.Init(); err != nil {
		log.Fatalf("failed to initialize IMU manager: %v", err)
		return err
	}

	// --- Choose orientation source (mock vs real IMU) ---
	useMock := false
	var mockSrc orientation.Source

	if useMock {
		log.Println("using mock orientation source")
		mockSrc = orientation.NewMockSource()
	} else {
		if imuManager.IsLeftIMUAvailable() {
			log.Println("using left IMU for orientation")
		} else {
			log.Println("WARNING: left IMU not available, orientation may be unreliable")
		}
	}

	// --- connect to MQTT ---
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDProducer)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
		return token.Error()
	}
	defer client.Disconnect(250)

	log.Println("connected to MQTT, starting publish loop")

	// Track previous pose and time for gyro integration
	var prevPose orientation.Pose
	var lastTickTime time.Time

	// Counter for per-second logging (log extra data every N ticks)
	tickCounter := 0
	logInterval := cfg.ConsoleLogInterval / cfg.IMUSampleInterval // Calculate ticks per log interval

	// main tick
	ticker := time.NewTicker(time.Duration(cfg.IMUSampleInterval) * time.Millisecond)
	defer ticker.Stop()

	for t := range ticker.C {
		tickCounter++
		// Calculate delta time for gyro integration
		var deltaTime float64
		if lastTickTime.IsZero() {
			deltaTime = 0.1 // First iteration, assume 100ms
		} else {
			deltaTime = t.Sub(lastTickTime).Seconds()
		}
		lastTickTime = t

		// 1) Orientation (raw and fused)
		var pose orientation.Pose
		var rawIMU imu_raw.IMURaw
		if useMock {
			var err error
			pose, err = mockSrc.Next()
			if err != nil {
				log.Printf("error from mock orientation source: %v", err)
				continue
			}
		} else {
			// Read raw IMU data from left IMU
			var err error
			rawIMU, err = imuManager.ReadLeftIMU()
			if err != nil {
				log.Printf("error reading left IMU: %v", err)
				continue
			}
			// Compute pose with gyro integration
			pose = orientation.ComputePoseFromIMURaw(
				float64(rawIMU.Ax),
				float64(rawIMU.Ay),
				float64(rawIMU.Az),
				float64(rawIMU.Gx),
				float64(rawIMU.Gy),
				float64(rawIMU.Gz),
				prevPose,
				deltaTime,
			)
		}

		// Update previous pose for next iteration
		prevPose = pose

		// Publish raw pose
		payload, err := json.Marshal(pose)
		if err != nil {
			log.Printf("json marshal error (pose): %v", err)
		} else {
			if token := client.Publish(cfg.TopicPose, 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (pose): %v", token.Error())
				continue
			}
			// fused pose (same for now)
			if token := client.Publish(cfg.TopicPoseFused, 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (pose/fused): %v", token.Error())
				continue
			}
		}

		// 2) Left/right IMU raw
		var imuL imu_raw.IMURaw
		if useMock {
			// In mock mode, create a dummy reading
			imuL = imu_raw.IMURaw{Source: "mock"}
		} else {
			// In real mode, we already read from left IMU for orientation
			imuL = rawIMU
		}

		if payload, err := json.Marshal(imuL); err != nil {
			log.Printf("left IMU marshal error: %v", err)
			continue
		} else {
			if token := client.Publish(cfg.TopicIMULeft, 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (imu/left): %v", token.Error())
				continue
			}
		}

		// --- MAG TEST/DEBUG: publish mag-only topic for left IMU ---
		{
			mn := magNorm(imuL.Mx, imuL.My, imuL.Mz)
			magTest := struct {
				Mx   int16   `json:"mx"`
				My   int16   `json:"my"`
				Mz   int16   `json:"mz"`
				Norm float64 `json:"norm"`
				Time string  `json:"time"`
			}{
				Mx:   imuL.Mx,
				My:   imuL.My,
				Mz:   imuL.Mz,
				Norm: mn,
				Time: t.Format(time.RFC3339),
			}

			if payload, err := json.Marshal(magTest); err != nil {
				log.Printf("mag marshal error: %v", err)
			} else {
				client.Publish(cfg.TopicMagLeft, 0, true, payload)
			}
		}

		// Read and publish right IMU
		if imuManager.IsRightIMUAvailable() {
			if imuR, err := imuManager.ReadRightIMU(); err != nil {
				log.Printf("right IMU read error: %v", err)
			} else if payload, err := json.Marshal(imuR); err != nil {
				log.Printf("right IMU marshal error: %v", err)
			} else {
				if token := client.Publish(cfg.TopicIMURight, 0, true, payload); token.Wait() && token.Error() != nil {
					log.Printf("MQTT publish error (imu/right): %v", token.Error())
				}

				// --- MAG TEST/DEBUG: publish mag-only topic for right IMU ---
				mn := magNorm(imuR.Mx, imuR.My, imuR.Mz)
				magTest := struct {
					Mx   int16   `json:"mx"`
					My   int16   `json:"my"`
					Mz   int16   `json:"mz"`
					Norm float64 `json:"norm"`
					Time string  `json:"time"`
				}{
					Mx:   imuR.Mx,
					My:   imuR.My,
					Mz:   imuR.Mz,
					Norm: mn,
					Time: t.Format(time.RFC3339),
				}

				if payload, err := json.Marshal(magTest); err != nil {
					log.Printf("right mag marshal error: %v", err)
				} else {
					client.Publish(cfg.TopicMagRight, 0, true, payload)
				}
			}
		}

		// 3) Left/right env (BMP)
		if envL, err := sensors.ReadLeftEnv(); err != nil {
			log.Printf("left env read error: %v", err)
			continue
		} else if payload, err := json.Marshal(envL); err != nil {
			log.Printf("left env marshal error: %v", err)
			continue
		} else {
			if token := client.Publish(cfg.TopicBMPLeft, 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (bmp/left): %v", token.Error())
				continue
			}
		}

		if envR, err := sensors.ReadRightEnv(); err != nil {
			log.Printf("right env read error: %v", err)
			continue
		} else if payload, err := json.Marshal(envR); err != nil {
			log.Printf("right env marshal error: %v", err)
			continue
		} else {
			if token := client.Publish(cfg.TopicBMPRight, 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (bmp/right): %v", token.Error())
				continue
			}
		}

		// --- Log all sensor data once per second ---
		if tickCounter >= logInterval {
			tickCounter = 0

			// Left IMU
			mn := magNorm(imuL.Mx, imuL.My, imuL.Mz)
			log.Printf("%s | pose R=%.2f P=%.2f Y=%.2f",
				t.Format(time.RFC3339),
				pose.Roll, pose.Pitch, pose.Yaw,
			)
			log.Printf("  [LEFT IMU] accel ax=%d ay=%d az=%d | gyro gx=%d gy=%d gz=%d | mag mx=%d my=%d mz=%d | |B|=%.1f",
				imuL.Ax, imuL.Ay, imuL.Az,
				imuL.Gx, imuL.Gy, imuL.Gz,
				imuL.Mx, imuL.My, imuL.Mz,
				mn,
			)

			// Right IMU
			if imuManager.IsRightIMUAvailable() {
				if imuR, err := imuManager.ReadRightIMU(); err == nil {
					mnR := magNorm(imuR.Mx, imuR.My, imuR.Mz)
					log.Printf("  [RIGHT IMU] accel ax=%d ay=%d az=%d | gyro gx=%d gy=%d gz=%d | mag mx=%d my=%d mz=%d | |B|=%.1f",
						imuR.Ax, imuR.Ay, imuR.Az,
						imuR.Gx, imuR.Gy, imuR.Gz,
						imuR.Mx, imuR.My, imuR.Mz,
						mnR,
					)
				}
			}

			// Left BMP
			if envL, err := sensors.ReadLeftEnv(); err == nil {
				log.Printf("  [LEFT BMP] temp=%.2f°C pressure=%.2fmbar / %.2fhPa", envL.Temperature, envL.PressureMbar, envL.PressureHPa)
			}

			// Right BMP
			if envR, err := sensors.ReadRightEnv(); err == nil {
				log.Printf("  [RIGHT BMP] temp=%.2f°C pressure=%.2fmbar / %.2fhPa", envR.Temperature, envR.PressureMbar, envR.PressureHPa)
			}
		}
	}
	return nil
}
