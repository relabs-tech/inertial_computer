// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
)

func main() {
	log.Println("starting inertial-computer web server (MQTT subscriber)")

	// Load configuration
	if err := config.InitGlobal("inertial_config.txt"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Println("Note: Calibration requires IMU producer to be running (sudo ./imu_producer)")

	if err := app.RunWeb(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
