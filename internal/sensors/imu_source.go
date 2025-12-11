package sensors

import (
	"fmt"

	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/devices/v3/mpu9250"
	"periph.io/x/host/v3"
)

// Left IMU is connected to SPI2 (/dev/spidev6.0) with CS on GPIO18,
// as in the legacy project.
const spiLeftIMU = "/dev/spidev6.0"
const csLeftIMUPin = "18"
// Right IMU defaults
const spiRightIMU = "/dev/spidev0.0"
const csRightIMUPin = "8"

type imuSource struct {
	imu *mpu9250.MPU9250
}

// NewIMUSourceLeft initializes the left MPU9250 over SPI and returns
// an IMURawReader that can read raw accelerometer, gyroscope, and magnetometer data.
func NewIMUSourceLeft() (IMURawReader, error) {
	// Initialize periph host once.
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("periph host init: %w", err)
	}

	cs := gpioreg.ByName(csLeftIMUPin)
	if cs == nil {
		return nil, fmt.Errorf("left IMU CS pin %q not found", csLeftIMUPin)
	}

	tr, err := mpu9250.NewSpiTransport(spiLeftIMU, cs)
	if err != nil {
		return nil, fmt.Errorf("left IMU SPI transport: %w", err)
	}

	imu, err := mpu9250.New(*tr)
	if err != nil {
		return nil, fmt.Errorf("left IMU new device: %w", err)
	}

	if err := imu.Init(); err != nil {
		return nil, fmt.Errorf("left IMU init: %w", err)
	}

	// Optional: self-test & calibrate at startup. You can comment these
	// out if they are too slow for dev.
	if _, err := imu.SelfTest(); err != nil {
		return nil, fmt.Errorf("left IMU self-test: %w", err)
	}
	if err := imu.Calibrate(); err != nil {
		return nil, fmt.Errorf("left IMU calibrate: %w", err)
	}

	// You can also set accel range here if needed, e.g. 2G:
	// _ = imu.SetAccelRange(byte(2))

	return &imuSource{imu: imu}, nil
}

	// NewIMUSourceRight initializes the right MPU9250 over SPI and returns
	// an IMURawReader for the right device.
	func NewIMUSourceRight() (IMURawReader, error) {
		// Initialize periph host once.
		if _, err := host.Init(); err != nil {
			return nil, fmt.Errorf("periph host init: %w", err)
		}

		cs := gpioreg.ByName(csRightIMUPin)
		if cs == nil {
			return nil, fmt.Errorf("right IMU CS pin %q not found", csRightIMUPin)
		}

		tr, err := mpu9250.NewSpiTransport(spiRightIMU, cs)
		if err != nil {
			return nil, fmt.Errorf("right IMU SPI transport: %w", err)
		}

		imu, err := mpu9250.New(*tr)
		if err != nil {
			return nil, fmt.Errorf("right IMU new device: %w", err)
		}

		if err := imu.Init(); err != nil {
			return nil, fmt.Errorf("right IMU init: %w", err)
		}

		// Optional self-test & calibrate
		if _, err := imu.SelfTest(); err != nil {
			return nil, fmt.Errorf("right IMU self-test: %w", err)
		}
		if err := imu.Calibrate(); err != nil {
			return nil, fmt.Errorf("right IMU calibrate: %w", err)
		}

		return &imuSource{imu: imu}, nil
	}

// IMURawReader is an interface for reading raw IMU data (accel, gyro, mag).
// Pose computation is decoupled and handled by pure functions like orientation.AccelToPose.
type IMURawReader interface {
	ReadRaw() (imu_raw.IMURaw, error)
}

// ReadRaw reads accelerometer, gyroscope, and magnetometer data from the left IMU
// and returns raw values. This is a pure data read with no pose computation.
func (s *imuSource) ReadRaw() (imu_raw.IMURaw, error) {
	ax, err := s.imu.GetAccelerationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU acc X: %w", err)
	}
	ay, err := s.imu.GetAccelerationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU acc Y: %w", err)
	}
	az, err := s.imu.GetAccelerationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU acc Z: %w", err)
	}

	// Read gyroscope (rotation) values from the MPU9250
	gx, err := s.imu.GetRotationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU gyro X: %w", err)
	}
	gy, err := s.imu.GetRotationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU gyro Y: %w", err)
	}
	gz, err := s.imu.GetRotationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("left IMU gyro Z: %w", err)
	}

	// TODO: read actual mag values from EXT_SENS_DATA registers
	mx := int16(0)
	my := int16(0)
	mz := int16(0)

	return imu_raw.IMURaw{
		Source: "left",
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
