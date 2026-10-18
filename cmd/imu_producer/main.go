// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/ledger"
	"github.com/relabs-tech/inertial_computer/internal/watchdog"
)

func main() {
	configPath := flag.String("config", "./inertial_config.txt", "path to configuration file")
	flag.Parse()

	log.Println("starting inertial-computer Inertial (IMU, BMP) producer (Inertial → MQTT)")

	// Load configuration
	if err := config.InitGlobal(*configPath); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ledger.Init(config.Get().LedgerFilePath)
	ledger.Log("imu_producer", "start", "", nil)
	watchdog.StartHeartbeat(context.Background(), 10*time.Second)

	if err := app.RunInertialProducer(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
