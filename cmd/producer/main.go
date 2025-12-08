package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
)

func main() {
	log.Println("starting inertial-computer Inertial (IMU, BMP) producer (Inertial → MQTT)")
	if err := app.RunInertialProducer(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
