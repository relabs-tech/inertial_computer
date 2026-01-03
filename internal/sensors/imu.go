// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package sensors

import (
	"fmt"
	"sync"

	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
)

// IMUManager manages persistent left and right IMU sensor instances.
// It initializes hardware once and provides thread-safe access to both IMUs.
type IMUManager struct {
	leftIMU     IMURawReader
	rightIMU    IMURawReader
	mu          sync.RWMutex
	initialized bool
}

var (
	defaultManager *IMUManager
	managerOnce    sync.Once
)

// GetIMUManager returns the singleton IMU manager instance.
func GetIMUManager() *IMUManager {
	managerOnce.Do(func() {
		defaultManager = &IMUManager{}
	})
	return defaultManager
}

// Init initializes both left and right IMU sensors.
// This should be called once at application startup.
// Returns error if both IMUs fail to initialize, but allows partial success.
func (m *IMUManager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	var leftErr, rightErr error

	// Initialize left IMU
	m.leftIMU, leftErr = NewIMUSourceLeft()
	if leftErr != nil {
		fmt.Printf("Warning: Left IMU initialization failed: %v\n", leftErr)
	} else {
		fmt.Println("Left IMU initialized successfully")
	}

	// Initialize right IMU
	m.rightIMU, rightErr = NewIMUSourceRight()
	if rightErr != nil {
		fmt.Printf("Warning: Right IMU initialization failed: %v\n", rightErr)
	} else {
		fmt.Println("Right IMU initialized successfully")
	}

	// Fail only if both IMUs failed
	if leftErr != nil && rightErr != nil {
		return fmt.Errorf("both IMUs failed to initialize: left=%v, right=%v", leftErr, rightErr)
	}

	m.initialized = true
	return nil
}

// ReadLeftIMU reads raw data from the left IMU sensor.
// Returns error if left IMU is not available.
func (m *IMUManager) ReadLeftIMU() (imu_raw.IMURaw, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU manager not initialized")
	}
	if m.leftIMU == nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU not available")
	}
	return m.leftIMU.ReadRaw()
}

// ReadRightIMU reads raw data from the right IMU sensor.
// Returns error if right IMU is not available.
func (m *IMUManager) ReadRightIMU() (imu_raw.IMURaw, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU manager not initialized")
	}
	if m.rightIMU == nil {
		return imu_raw.IMURaw{}, fmt.Errorf("right IMU not available")
	}
	return m.rightIMU.ReadRaw()
}

// IsLeftIMUAvailable returns true if the left IMU is initialized and available.
func (m *IMUManager) IsLeftIMUAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized && m.leftIMU != nil
}

// IsRightIMUAvailable returns true if the right IMU is initialized and available.
func (m *IMUManager) IsRightIMUAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized && m.rightIMU != nil
}
