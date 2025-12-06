package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
)

func main() {
	log.Println("starting inertial-computer GPS producer (NMEA → MQTT)")

	if err := app.RunGPSProducer(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
