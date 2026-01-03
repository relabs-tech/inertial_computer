// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package main

import (
	"log"
	"net/http"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

func main() {
	log.Println("starting MPU9250 register debug tool (standalone)")

	if err := config.InitGlobal("inertial_config.txt"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Println("Initializing IMU manager...")
	imuManager := sensors.GetIMUManager()
	if err := imuManager.Init(); err != nil {
		log.Printf("Warning: IMU initialization had issues: %v", err)
		log.Println("Continuing anyway - at least one IMU may be available")
	}

	if imuManager.IsLeftIMUAvailable() {
		log.Println("Left IMU available")
	} else {
		log.Println("Warning: Left IMU not available")
	}

	if imuManager.IsRightIMUAvailable() {
		log.Println("Right IMU available")
	} else {
		log.Println("Warning: Right IMU not available")
	}

	http.HandleFunc("/ws", app.HandleRegisterDebugWS)

	// API endpoint for live IMU data
	http.HandleFunc("/api/imu", app.HandleIMUData)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/register_debug.html")
	})

	addr := ":8081"
	log.Printf("Register debug tool listening on %s", addr)
	log.Printf("Open http://localhost:8081 in your browser")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
