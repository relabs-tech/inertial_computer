package app

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

func RunInertialProducer() error {
	log.Println("starting inertial-computer orientation/env producer")

	// --- choose orientation source (mock vs IMU) ---
	useMock := false

	var (
		imuReader sensors.IMURawReader
		mockSrc   orientation.Source
		err       error
	)

	if useMock {
		log.Println("using mock orientation source")
		mockSrc = orientation.NewMockSource()
	} else {
		log.Println("using LEFT IMU raw reader")
		imuReader, err = sensors.NewIMUSourceLeft()
		if err != nil {
			log.Fatalf("failed to initialize IMU source: %v", err)
			return err
		}
	}

	// --- connect to MQTT ---
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-main-producer")

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

	// main tick
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for t := range ticker.C {
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
			// Read raw IMU data and compute pose with gyro integration
			var err error
			rawIMU, err = imuReader.ReadRaw()
			if err != nil {
				log.Printf("error reading raw IMU: %v", err)
				continue
			}
			// Convert int16 to float64 for pose computation
			// Use gyro integration to get yaw from angular velocity
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
			if token := client.Publish("inertial/pose", 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (pose): %v", token.Error())
				continue
			}
			// fused pose (same for now)
			if token := client.Publish("inertial/pose/fused", 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (pose/fused): %v", token.Error())
				continue
			}
		}

		// 2) Left/right IMU raw
		var imuL imu_raw.IMURaw
		if useMock {
			var err error
			imuL, err = sensors.ReadLeftIMURaw()
			if err != nil {
				log.Printf("left IMU read error: %v", err)
				continue
			}
		} else {
			// we already read rawIMU above from the real IMU
			imuL = rawIMU
		}

		if payload, err := json.Marshal(imuL); err != nil {
			log.Printf("left IMU marshal error: %v", err)
			continue
		} else {
			if token := client.Publish("inertial/imu/left", 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (imu/left): %v", token.Error())
				continue
			}
		}

		if imuR, err := sensors.ReadRightIMURaw(); err != nil {
			log.Printf("right IMU read error: %v", err)
			continue
		} else if payload, err := json.Marshal(imuR); err != nil {
			log.Printf("right IMU marshal error: %v", err)
			continue
		} else {
			if token := client.Publish("inertial/imu/right", 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (imu/right): %v", token.Error())
				continue
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
			if token := client.Publish("inertial/bmp/left", 0, true, payload); token.Wait() && token.Error() != nil {
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
			if token := client.Publish("inertial/bmp/right", 0, true, payload); token.Wait() && token.Error() != nil {
				log.Printf("MQTT publish error (bmp/right): %v", token.Error())
				continue
			}
		}

		log.Printf("%s tick: pose R=%.2f P=%.2f Y=%.2f | left accel ax=%d ay=%d az=%d | left gyro gx=%d gy=%d gz=%d", t.Format(time.RFC3339), pose.Roll, pose.Pitch, pose.Yaw, imuL.Ax, imuL.Ay, imuL.Az, imuL.Gx, imuL.Gy, imuL.Gz)
	}
	return nil
}
