// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package orientation

import (
	"math"
	"time"
)

type mockSource struct {
	start time.Time
}

// NewMockSource creates a mock orientation source that
// generates smooth changing values.
func NewMockSource() Source {
	return &mockSource{start: time.Now()}
}

func (m *mockSource) Next() (Pose, error) {
	elapsed := time.Since(m.start).Seconds()

	return Pose{
		Roll:  20 * math.Sin(elapsed),
		Pitch: 15 * math.Cos(elapsed*0.7),
		Yaw:   math.Mod(elapsed*30, 360),
	}, nil
}
