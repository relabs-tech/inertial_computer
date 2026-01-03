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

// ReadRegister reads a single register from the specified IMU.
// imuID should be "left" or "right".
func (m *IMUManager) ReadRegister(imuID string, regAddr byte) (byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return 0, fmt.Errorf("IMU manager not initialized")
	}

	var imuSrc *imuSource
	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return 0, fmt.Errorf("left IMU not available")
		}
		imuSrc = m.leftIMU.(*imuSource)
	case "right":
		if m.rightIMU == nil {
			return 0, fmt.Errorf("right IMU not available")
		}
		imuSrc = m.rightIMU.(*imuSource)
	default:
		return 0, fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	return imuSrc.imu.ReadRegister(regAddr)
}

// WriteRegister writes a single register to the specified IMU.
// imuID should be "left" or "right".
func (m *IMUManager) WriteRegister(imuID string, regAddr byte, value byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("IMU manager not initialized")
	}

	var imuSrc *imuSource
	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return fmt.Errorf("left IMU not available")
		}
		imuSrc = m.leftIMU.(*imuSource)
	case "right":
		if m.rightIMU == nil {
			return fmt.Errorf("right IMU not available")
		}
		imuSrc = m.rightIMU.(*imuSource)
	default:
		return fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	return imuSrc.imu.WriteRegister(regAddr, value)
}

// ReadAllRegisters reads all MPU9250 registers (0x00-0x7F) from the specified IMU.
func (m *IMUManager) ReadAllRegisters(imuID string) (map[byte]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("IMU manager not initialized")
	}

	var imuSrc *imuSource
	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return nil, fmt.Errorf("left IMU not available")
		}
		imuSrc = m.leftIMU.(*imuSource)
	case "right":
		if m.rightIMU == nil {
			return nil, fmt.Errorf("right IMU not available")
		}
		imuSrc = m.rightIMU.(*imuSource)
	default:
		return nil, fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	registers := make(map[byte]byte)
	for addr := byte(0x00); addr <= 0x7F; addr++ {
		value, err := imuSrc.imu.ReadRegister(addr)
		if err != nil {
			return nil, fmt.Errorf("error reading register 0x%02X: %w", addr, err)
		}
		registers[addr] = value
	}

	return registers, nil
}

// SetSPISpeed sets the SPI read and write speeds for the specified IMU.
// TODO: Implement SPI speed control in mpu9250 driver
func (m *IMUManager) SetSPISpeed(imuID string, readSpeed, writeSpeed int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("IMU manager not initialized")
	}

	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return fmt.Errorf("left IMU not available")
		}
	case "right":
		if m.rightIMU == nil {
			return fmt.Errorf("right IMU not available")
		}
	default:
		return fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	// TODO: Call driver method when implemented
	return fmt.Errorf("SPI speed control not yet implemented")
}

// GetSPISpeed gets the current SPI read and write speeds for the specified IMU.
// TODO: Implement SPI speed query in mpu9250 driver
func (m *IMUManager) GetSPISpeed(imuID string) (readSpeed, writeSpeed int64, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return 0, 0, fmt.Errorf("IMU manager not initialized")
	}

	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return 0, 0, fmt.Errorf("left IMU not available")
		}
	case "right":
		if m.rightIMU == nil {
			return 0, 0, fmt.Errorf("right IMU not available")
		}
	default:
		return 0, 0, fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	// TODO: Call driver method when implemented
	return 0, 0, fmt.Errorf("SPI speed query not yet implemented")
}

// GetRegisterMap returns metadata for all MPU9250 registers.
func (m *IMUManager) GetRegisterMap() []RegisterInfo {
	return getMPU9250RegisterMap()
}

// ReinitializeIMU closes and reopens the SPI connection for the specified IMU.
func (m *IMUManager) ReinitializeIMU(imuID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("IMU manager not initialized")
	}

	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return fmt.Errorf("left IMU not available")
		}
		// Reinitialize left IMU
		newIMU, err := NewIMUSourceLeft()
		if err != nil {
			return fmt.Errorf("failed to reinitialize left IMU: %w", err)
		}
		m.leftIMU = newIMU
		fmt.Println("Left IMU reinitialized successfully")
		return nil

	case "right":
		if m.rightIMU == nil {
			return fmt.Errorf("right IMU not available")
		}
		// Reinitialize right IMU
		newIMU, err := NewIMUSourceRight()
		if err != nil {
			return fmt.Errorf("failed to reinitialize right IMU: %w", err)
		}
		m.rightIMU = newIMU
		fmt.Println("Right IMU reinitialized successfully")
		return nil

	default:
		return fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}
}

// ApplyRegisterConfig loads a register configuration JSON file and applies it to the specified IMU.
// TODO: Implement proper JSON loading and validation
func (m *IMUManager) ApplyRegisterConfig(imuID, configFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("IMU manager not initialized")
	}

	var imuSrc *imuSource
	switch imuID {
	case "left":
		if m.leftIMU == nil {
			return fmt.Errorf("left IMU not available")
		}
		imuSrc = m.leftIMU.(*imuSource)
	case "right":
		if m.rightIMU == nil {
			return fmt.Errorf("right IMU not available")
		}
		imuSrc = m.rightIMU.(*imuSource)
	default:
		return fmt.Errorf("invalid IMU ID: %s (must be 'left' or 'right')", imuID)
	}

	// TODO: Load JSON file, parse, validate, and apply register writes
	_ = imuSrc // Use variable to avoid compiler warning
	_ = configFile
	return fmt.Errorf("register config loading not yet implemented")
}

// ExportRegisterConfig reads all registers from the specified IMU.
// This is an alias for ReadAllRegisters used by the export functionality.
func (m *IMUManager) ExportRegisterConfig(imuID string) (map[byte]byte, error) {
	return m.ReadAllRegisters(imuID)
}

// RegisterInfo holds metadata about a single MPU9250 register.
// RegisterInfo holds metadata about a single MPU9250 register.
type RegisterInfo struct {
	Address     string     `json:"address"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Access      string     `json:"access"` // "R", "W", "RW"
	Default     string     `json:"default,omitempty"`
	BitFields   []BitField `json:"bit_fields,omitempty"`
}

// BitField describes a single bit or bit range within a register
type BitField struct {
	Bits        string `json:"bits"`        // e.g., "7:6", "5", "4:3"
	Name        string `json:"name"`        // e.g., "FS_SEL", "DLPF_CFG"
	Description string `json:"description"` // Function description
	Values      string `json:"values"`      // Possible values
}
