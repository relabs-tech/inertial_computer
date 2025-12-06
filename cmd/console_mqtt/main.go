package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
)

func main() {
	log.Println("starting inertial-computer console (MQTT subscriber)")

	if err := app.RunConsoleMQTT(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
