package main

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func main() {
	log.Println("starting inertial-computer orientation producer")

	// ---- choose data source ----
	useMock := false // set to true if you want to go back to mock

	var (
		src orientation.Source
		err error
	)

	if useMock {
		log.Println("using mock orientation source")
		src = orientation.NewMockSource()
	} else {
		log.Println("using LEFT IMU orientation source")
		src, err = orientation.NewIMUSourceLeft()
		if err != nil {
			log.Fatalf("failed to initialize IMU source: %v", err)
		}
	}

	// ---- connect to MQTT ----
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-orientation-producer")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	defer client.Disconnect(250)

	log.Println("connected to MQTT, starting publish loop")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for t := range ticker.C {
		pose, err := src.Next()
		if err != nil {
			log.Printf("error from orientation source: %v", err)
			continue
		}

		payload, err := json.Marshal(pose)
		if err != nil {
			log.Printf("json marshal error: %v", err)
			continue
		}

		token := client.Publish("inertial/pose", 0, true, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("MQTT publish error: %v", token.Error())
			continue
		}

		log.Printf("%s published pose: %+v", t.Format(time.RFC3339), pose)
	}
}
