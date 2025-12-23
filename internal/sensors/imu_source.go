package sensors

import (
	"fmt"
	"log"

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

	// Self-test
	testResult, err := imu.SelfTest()
	if err != nil {
		log.Printf("Warning: %s IMU self-test failed: %v", name, err)
	} else {
		log.Printf("%s IMU self-test passed: %+v", name, testResult)
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
