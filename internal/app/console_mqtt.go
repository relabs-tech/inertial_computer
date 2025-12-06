package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunConsoleMQTT() error {
	// 1) Connect to MQTT broker on the Pi
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-console-subscriber")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Println("console connected to MQTT broker at tcp://localhost:1883")

	// 2) Subscribe to the pose topic and print every message
	token := client.Subscribe("inertial/pose", 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("MQTT payload unmarshal error: %v", err)
			return
		}

		fmt.Printf(
			"ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			p.Roll, p.Pitch, p.Yaw,
		)
	})
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	log.Println("console subscribed to MQTT topic inertial/pose")

	// 3) Block until Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("console shutting down")
	client.Disconnect(250)
	return nil
}
