package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/gps"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunConsoleMQTT() error {
	cfg := config.Get()

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDConsole)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("console: connected to MQTT broker at %s", cfg.MQTTBroker)

	// Subscribe to left pose
	poseLeftToken := client.Subscribe(cfg.TopicPoseLeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("console: left pose unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[LEFT]  ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			p.Roll, p.Pitch, p.Yaw,
		)
	})
	poseLeftToken.Wait()
	if poseLeftToken.Error() != nil {
		return poseLeftToken.Error()
	}
	log.Printf("console: subscribed to %s", cfg.TopicPoseLeft)

	// Subscribe to right pose
	poseRightToken := client.Subscribe(cfg.TopicPoseRight, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("console: right pose unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[RIGHT] ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			p.Roll, p.Pitch, p.Yaw,
		)
	})
	poseRightToken.Wait()
	if poseRightToken.Error() != nil {
		return poseRightToken.Error()
	}
	log.Printf("console: subscribed to %s", cfg.TopicPoseRight)

	// Subscribe to fused orientation
	fusedToken := client.Subscribe(cfg.TopicPoseFused, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("console: fused pose unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[FUSE] ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			p.Roll, p.Pitch, p.Yaw,
		)
	})
	fusedToken.Wait()
	if fusedToken.Error() != nil {
		return fusedToken.Error()
	}
	log.Printf("console: subscribed to %s", cfg.TopicPoseFused)

	// Subscribe to IMU left
	imuLeftToken := client.Subscribe(cfg.TopicIMULeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s imu_raw.IMURaw
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("console: imu left unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[IMU-L] ax=%6d ay=%6d az=%6d  gx=%6d gy=%6d gz=%6d  mx=%6d my=%6d mz=%6d\n",
			s.Ax, s.Ay, s.Az, s.Gx, s.Gy, s.Gz, s.Mx, s.My, s.Mz,
		)
	})
	imuLeftToken.Wait()
	if imuLeftToken.Error() != nil {
		return imuLeftToken.Error()
	}

	// Subscribe to IMU right
	imuRightToken := client.Subscribe(cfg.TopicIMURight, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s imu_raw.IMURaw
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("console: imu right unmarshal error: %v", err)
			return
		}
		fmt.Printf(
			"[IMU-R] ax=%6d ay=%6d az=%6d  gx=%6d gy=%6d gz=%6d  mx=%6d my=%6d mz=%6d\n",
			s.Ax, s.Ay, s.Az, s.Gx, s.Gy, s.Gz, s.Mx, s.My, s.Mz,
		)
	})

	imuRightToken.Wait()
	if imuRightToken.Error() != nil {
		return imuRightToken.Error()
	}

	log.Printf("console: subscribed to %s", cfg.TopicIMURight)

	// Subscribe to GPS
	gpsToken := client.Subscribe(cfg.TopicGPS, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var f gps.Fix
		if err := json.Unmarshal(msg.Payload(), &f); err != nil {
			log.Printf("console: gps unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[GPS ]  time=%s date=%s lat=%.6f lon=%.6f speed=%.1fkn course=%.1f° validity=%s\n",
			f.Time, f.Date, f.Latitude, f.Longitude, f.SpeedKnots, f.CourseDeg, f.Validity,
		)
	})
	gpsToken.Wait()
	if gpsToken.Error() != nil {
		return gpsToken.Error()
	}
	log.Printf("console: subscribed to %s", cfg.TopicGPS)

	// Wait for Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("console: shutting down")
	client.Disconnect(250)
	return nil
}
