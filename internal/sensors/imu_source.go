// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package sensors

import (
	"fmt"
	"log"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/config"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/devices/v3/mpu9250"
	"periph.io/x/host/v3"
)

// IMURawReader defines the interface for reading raw IMU data.
type IMURawReader interface {
	ReadRaw() (imu_raw.IMURaw, error)
}

type imuSource struct {
	name     string // "left" or "right" for logging
	imu      *mpu9250.MPU9250
	magCal   *mpu9250.MagCal
	magReady bool
}

// NewIMUSourceLeft initializes the left MPU9250 over SPI.
func NewIMUSourceLeft() (IMURawReader, error) {
	cfg := config.Get()
	return newIMUSource("left", cfg.IMULeftSPIDevice, cfg.IMULeftCSPin)
}

// NewIMUSourceRight initializes the right MPU9250 over SPI.
func NewIMUSourceRight() (IMURawReader, error) {
	cfg := config.Get()
	return newIMUSource("right", cfg.IMURightSPIDevice, cfg.IMURightCSPin)
}

// newIMUSource is a unified initialization function for both left and right IMUs.
func newIMUSource(name, spiDev, csPin string) (IMURawReader, error) {
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("%s IMU: periph host init: %w", name, err)
	}

	cs := gpioreg.ByName(csPin)
	if cs == nil {
		return nil, fmt.Errorf("%s IMU: CS pin %q not found", name, csPin)
	}

	tr, err := mpu9250.NewSpiTransport(spiDev, cs)
	if err != nil {
		return nil, fmt.Errorf("%s IMU: SPI transport (%s): %w", name, spiDev, err)
	}

	imu, err := mpu9250.New(tr)
	if err != nil {
		return nil, fmt.Errorf("%s IMU: device creation: %w", name, err)
	}

	if err := imu.Init(); err != nil {
		return nil, fmt.Errorf("%s IMU: initialization: %w", name, err)
	}

	// Apply configured sensor ranges
	cfg := config.Get()
	if err := imu.SetAccelRange(cfg.IMUAccelRange); err != nil {
		return nil, fmt.Errorf("%s IMU: set accel range: %w", name, err)
	}
	log.Printf("%s IMU: accelerometer range set to %d (±%dg)", name, cfg.IMUAccelRange, []int{2, 4, 8, 16}[cfg.IMUAccelRange])

	if err := imu.SetGyroRange(cfg.IMUGyroRange); err != nil {
		return nil, fmt.Errorf("%s IMU: set gyro range: %w", name, err)
	}
	log.Printf("%s IMU: gyroscope range set to %d (±%d°/s)", name, cfg.IMUGyroRange, []int{250, 500, 1000, 2000}[cfg.IMUGyroRange])

	// Configure sample rate
	if err := imu.SetDLPFMode(cfg.IMUDLPFConfig); err != nil {
		return nil, fmt.Errorf("%s IMU: set DLPF config: %w", name, err)
	}
	log.Printf("%s IMU: DLPF config set to %d", name, cfg.IMUDLPFConfig)

	if err := imu.SetSampleRateDivider(cfg.IMUSampleRateDiv); err != nil {
		return nil, fmt.Errorf("%s IMU: set sample rate divider: %w", name, err)
	}
	internalRate := 1000 // 1kHz for DLPF modes 0-6
	if cfg.IMUDLPFConfig == 7 {
		internalRate = 8000 // 8kHz when DLPF disabled
	}
	outputRate := internalRate / (1 + int(cfg.IMUSampleRateDiv))
	log.Printf("%s IMU: sample rate divider set to %d (output rate: %d Hz)", name, cfg.IMUSampleRateDiv, outputRate)

	if err := imu.SetAccelDLPF(cfg.IMUAccelDLPF); err != nil {
		return nil, fmt.Errorf("%s IMU: set accel DLPF: %w", name, err)
	}
	log.Printf("%s IMU: accelerometer DLPF set to %d", name, cfg.IMUAccelDLPF)

	// Self-test
	testResult, err := imu.SelfTest()
	if err != nil {
		log.Printf("Warning: %s IMU self-test failed: %v", name, err)
	} else {
		log.Printf("%s IMU self-test passed:", name)
		log.Printf("  Accelerometer deviation: X: %.2f%%, Y: %.2f%%, Z: %.2f%%",
			testResult.AccelDeviation.X, testResult.AccelDeviation.Y, testResult.AccelDeviation.Z)
		log.Printf("  Gyroscope deviation: X: %.2f%%, Y: %.2f%%, Z: %.2f%%",
			testResult.GyroDeviation.X, testResult.GyroDeviation.Y, testResult.GyroDeviation.Z)
	}

	// Calibration
	if err := imu.Calibrate(); err != nil {
		log.Printf("Warning: %s IMU calibration failed: %v", name, err)
	} else {
		log.Printf("%s IMU calibration complete", name)
	}

	// Magnetometer initialization (non-fatal)
	magCal, err := imu.InitMag()
	if err != nil {
		log.Printf("%s IMU: magnetometer initialization failed (will continue without mag): %v", name, err)
		return &imuSource{
			name:     name,
			imu:      imu,
			magReady: false,
		}, nil
	}

	log.Printf("%s IMU: magnetometer initialized successfully", name)
	return &imuSource{
		name:     name,
		imu:      imu,
		magCal:   magCal,
		magReady: true,
	}, nil
}

// ReadRaw reads accelerometer, gyroscope, and magnetometer data from this IMU.
func (s *imuSource) ReadRaw() (imu_raw.IMURaw, error) {
	// Read accelerometer
	ax, err := s.imu.GetAccelerationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU accel X: %w", s.name, err)
	}
	ay, err := s.imu.GetAccelerationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU accel Y: %w", s.name, err)
	}
	az, err := s.imu.GetAccelerationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU accel Z: %w", s.name, err)
	}

	// Read gyroscope
	gx, err := s.imu.GetRotationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU gyro X: %w", s.name, err)
	}
	gy, err := s.imu.GetRotationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU gyro Y: %w", s.name, err)
	}
	gz, err := s.imu.GetRotationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("%s IMU gyro Z: %w", s.name, err)
	}

	// Read magnetometer (if available)
	var mx, my, mz int16
	if s.magReady {
		mag, err := s.imu.ReadMag(s.magCal)
		if err != nil {
			log.Printf("%s IMU: magnetometer read error: %v", s.name, err)
		} else if mag.Overflow {
			log.Printf("%s IMU: magnetometer overflow detected", s.name)
		} else {
			// Store scaled µT values as int16 (multiply by 10 for precision)
			mx = int16(mag.X * 10)
			my = int16(mag.Y * 10)
			mz = int16(mag.Z * 10)
		}
	}

	return imu_raw.IMURaw{
		Source: s.name,
		Ax:     ax,
		Ay:     ay,
		Az:     az,
		Gx:     gx,
		Gy:     gy,
		Gz:     gz,
		Mx:     mx,
		My:     my,
		Mz:     mz,
	}, nil
}

// ReadAK8963Register reads a single register from the AK8963 magnetometer via I2C master.
// The AK8963 is accessed through MPU9250's internal I2C master using EXT_SENS_DATA.
func (s *imuSource) ReadAK8963Register(regAddr byte) (byte, error) {
	const AK8963_ADDR = 0x0C
	const I2C_SLV0_ADDR = 0x25
	const I2C_SLV0_REG = 0x26
	const I2C_SLV0_CTRL = 0x27
	const EXT_SENS_DATA_00 = 0x49

	// Configure I2C slave 0 for reading from AK8963
	// Set slave address with read bit (0x80)
	if err := s.imu.WriteRegister(I2C_SLV0_ADDR, AK8963_ADDR|0x80); err != nil {
		return 0, fmt.Errorf("failed to set AK8963 slave address: %w", err)
	}

	// Set register address to read from
	if err := s.imu.WriteRegister(I2C_SLV0_REG, regAddr); err != nil {
		return 0, fmt.Errorf("failed to set AK8963 register address: %w", err)
	}

	// Enable slave 0, read 1 byte
	if err := s.imu.WriteRegister(I2C_SLV0_CTRL, 0x81); err != nil {
		return 0, fmt.Errorf("failed to enable AK8963 read: %w", err)
	}

	// Wait for I2C transaction to complete
	time.Sleep(2 * time.Millisecond)

	// Read result from EXT_SENS_DATA_00
	value, err := s.imu.ReadRegister(EXT_SENS_DATA_00)
	if err != nil {
		return 0, fmt.Errorf("failed to read EXT_SENS_DATA_00: %w", err)
	}

	return value, nil
}

// WriteAK8963Register writes a single register to the AK8963 magnetometer via I2C master.
func (s *imuSource) WriteAK8963Register(regAddr byte, value byte) error {
	const AK8963_ADDR = 0x0C
	const I2C_SLV0_ADDR = 0x25
	const I2C_SLV0_REG = 0x26
	const I2C_SLV0_DO = 0x28
	const I2C_SLV0_CTRL = 0x27

	// Configure I2C slave 0 for writing to AK8963
	// Set slave address without read bit (0x00 for write)
	if err := s.imu.WriteRegister(I2C_SLV0_ADDR, AK8963_ADDR); err != nil {
		return fmt.Errorf("failed to set AK8963 slave address: %w", err)
	}

	// Set register address to write to
	if err := s.imu.WriteRegister(I2C_SLV0_REG, regAddr); err != nil {
		return fmt.Errorf("failed to set AK8963 register address: %w", err)
	}

	// Set data to write
	if err := s.imu.WriteRegister(I2C_SLV0_DO, value); err != nil {
		return fmt.Errorf("failed to set AK8963 write data: %w", err)
	}

	// Enable slave 0, write 1 byte
	if err := s.imu.WriteRegister(I2C_SLV0_CTRL, 0x81); err != nil {
		return fmt.Errorf("failed to enable AK8963 write: %w", err)
	}

	// Wait for I2C transaction to complete
	time.Sleep(2 * time.Millisecond)

	return nil
}

// ReadAllAK8963Registers reads all AK8963 registers (0x00-0x12).
func (s *imuSource) ReadAllAK8963Registers() (map[byte]byte, error) {
	registers := make(map[byte]byte)

	// Read accessible AK8963 registers
	for _, addr := range []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x10, 0x11, 0x12} {
		value, err := s.ReadAK8963Register(addr)
		if err != nil {
			return nil, fmt.Errorf("error reading AK8963 register 0x%02X: %w", addr, err)
		}
		registers[addr] = value
	}

	return registers, nil
}

// ReadRegister reads a single register from this IMU.
func (s *imuSource) ReadRegister(regAddr byte) (byte, error) {
	return s.imu.ReadRegister(regAddr)
}

// WriteRegister writes a single register to this IMU.
func (s *imuSource) WriteRegister(regAddr byte, value byte) error {
	return s.imu.WriteRegister(regAddr, value)
}

// ReadAllRegisters reads all MPU9250 registers (0x00-0x7F) from this IMU.
func (s *imuSource) ReadAllRegisters() (map[byte]byte, error) {
	registers := make(map[byte]byte)
	for addr := byte(0x00); addr <= 0x7F; addr++ {
		value, err := s.imu.ReadRegister(addr)
		if err != nil {
			return nil, fmt.Errorf("error reading register 0x%02X: %w", addr, err)
		}
		registers[addr] = value
	}
	return registers, nil
}
