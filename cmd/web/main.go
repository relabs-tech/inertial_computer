package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

func main() {
	log.Println("starting inertial-computer web server (MQTT subscriber, mock data)")

	// Load configuration
	if err := config.InitGlobal("inertial_config.txt"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize IMU manager for calibration support
	if err := sensors.GetIMUManager().Init(); err != nil {
		log.Printf("warning: failed to initialize IMU manager: %v (calibration will not be available)", err)
	} else {
		log.Println("IMU manager initialized successfully")
	}

	if err := app.RunWeb(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
