// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package app

import (
	"fmt"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/orientation" // adjust to your module path
)

func RunMockConsole() error {
	src := orientation.NewMockSource()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		pose, err := src.Next()
		if err != nil {
			return err
		}

		fmt.Printf(
			"ROLL=%6.2f  PITCH=%6.2f  YAW=%6.2f\n",
			pose.Roll,
			pose.Pitch,
			pose.Yaw,
		)
	}
	return nil
}
