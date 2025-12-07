package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/gps"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunConsoleMQTT() error {
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-console-subscriber")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Println("console: connected to MQTT broker at tcp://localhost:1883")

	// Subscribe to orientation
	poseToken := client.Subscribe("inertial/pose", 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("console: pose unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"[POSE]  ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			p.Roll, p.Pitch, p.Yaw,
		)
	})
	poseToken.Wait()
	if poseToken.Error() != nil {
		return poseToken.Error()
	}
	log.Println("console: subscribed to inertial/pose")

	// Subscribe to GPS
	gpsToken := client.Subscribe("inertial/gps", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("console: subscribed to inertial/gps")

	// Wait for Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("console: shutting down")
	client.Disconnect(250)
	return nil
}
