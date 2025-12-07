package orientation

import (
	"fmt"
	"math"

	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/devices/v3/mpu9250"
	"periph.io/x/host/v3"
)

// Left IMU is connected to SPI2 (/dev/spidev6.0) with CS on GPIO18,
// as in the legacy project.
const spiLeftIMU = "/dev/spidev6.0"
const csLeftIMUPin = "18"

type imuSource struct {
	imu *mpu9250.MPU9250
}

// NewIMUSourceLeft initializes the left MPU9250 over SPI and returns
// an orientation.Source that reads roll/pitch from the accelerometer.
// Yaw is currently set to 0 until we fuse the magnetometer.
func NewIMUSourceLeft() (Source, error) {
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

// Next reads accelerometer data from the IMU and computes roll/pitch
// using a simple accelerometer-only tilt estimate. Yaw is left at 0
// until proper fusion with gyro + magnetometer is implemented.
func (s *imuSource) Next() (Pose, error) {
	ax, err := s.imu.GetAccelerationX()
	if err != nil {
		return Pose{}, fmt.Errorf("left IMU acc X: %w", err)
	}
	ay, err := s.imu.GetAccelerationY()
	if err != nil {
		return Pose{}, fmt.Errorf("left IMU acc Y: %w", err)
	}
	az, err := s.imu.GetAccelerationZ()
	if err != nil {
		return Pose{}, fmt.Errorf("left IMU acc Z: %w", err)
	}

	// Convert to float64 for math. We don't need physical units to
	// get roll/pitch, only relative ratios.
	fx := float64(ax)
	fy := float64(ay)
	fz := float64(az)

	// Basic tilt estimation from accelerometer:
	// roll  = atan2(ay, az)
	// pitch = atan2(-ax, sqrt(ay^2 + az^2))
	rollRad := math.Atan2(fy, fz)
	pitchRad := math.Atan2(-fx, math.Sqrt(fy*fy+fz*fz))

	rollDeg := rollRad * 180.0 / math.Pi
	pitchDeg := pitchRad * 180.0 / math.Pi

	return Pose{
		Roll:  rollDeg,
		Pitch: pitchDeg,
		Yaw:   0, // placeholder; to be replaced with fused yaw later
	}, nil
}
