package main

import (
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func main() {
	log.Println("starting inertial-computer MQTT producer (mock)")

	// 1) Connect to MQTT broker on the Pi
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-producer-mock")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	defer client.Disconnect(250)

	src := orientation.NewMockSource()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for t := range ticker.C {
		pose, err := src.Next()
		if err != nil {
			log.Printf("error from mock source: %v", err)
			continue
		}

		payload, err := json.Marshal(pose)
		if err != nil {
			log.Printf("json marshal error: %v", err)
			continue
		}

		token := client.Publish("inertial/pose", 0, true, payload)
		token.Wait()

		log.Printf("%s published pose: %+v", t.Format(time.RFC3339), pose)
	}
}
