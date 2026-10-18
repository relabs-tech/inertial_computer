// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package main

import (
	"context"
	"log"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/app"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/ledger"
	"github.com/relabs-tech/inertial_computer/internal/watchdog"
)

func main() {
	log.Println("starting inertial-computer GPS producer (NMEA → MQTT)")

	// Load configuration
	if err := config.InitGlobal("inertial_config.txt"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ledger.Init(config.Get().LedgerFilePath)
	ledger.Log("gps_producer", "start", "", nil)
	watchdog.StartHeartbeat(context.Background(), 10*time.Second)

	if err := app.RunGPSProducer(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
