package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
)

func main() {
	log.Println("starting inertial-computer web server (MQTT subscriber, mock data)")

	if err := app.RunWeb(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
