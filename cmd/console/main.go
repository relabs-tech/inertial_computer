// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package main

import (
	"log"

	"github.com/relabs-tech/inertial_computer/internal/app"
)

func main() {
	log.Println("starting inertial-computer (mock console)")

	if err := app.RunMockConsole(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
