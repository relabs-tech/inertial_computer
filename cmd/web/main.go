package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
)

func main() {
	log.Println("starting inertial-computer web server (MQTT subscriber, mock data)")

	// Load configuration
	if err := config.InitGlobal("inertial_config.txt"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := app.RunWeb(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
