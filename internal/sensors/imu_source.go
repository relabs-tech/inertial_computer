package sensors

import (
	"fmt"

	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/devices/v3/mpu9250"
	"periph.io/x/host/v3"
)

// Left IMU is connected to SPI2 (/dev/spidev6.0) with CS on GPIO18.
const spiLeftIMU = "/dev/spidev6.0"
const csLeftIMUPin = "18"

// Right IMU defaults
const spiRightIMU = "/dev/spidev0.0"
const csRightIMUPin = "8"

type imuSource struct {
	imu      *mpu9250.MPU9250
	magCal   *mpu9250.MagCal
	magReady bool
}

// NewIMUSourceLeft initializes the left MPU9250 over SPI.
func NewIMUSourceLeft() (IMURawReader, error) {
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
	imu, err := mpu9250.New(tr)

	if err != nil {
		return nil, fmt.Errorf("left IMU new device: %w", err)
	}

	if err := imu.Init(); err != nil {
		return nil, fmt.Errorf("left IMU init: %w", err)
	}

	// Optional self-test & calibration
	if _, err := imu.SelfTest(); err != nil {
		return nil, fmt.Errorf("left IMU self-test: %w", err)
	}
	if err := imu.Calibrate(); err != nil {
		return nil, fmt.Errorf("left IMU calibrate: %w", err)
	}

	// --- Magnetometer init (non-fatal) ---
	magCal, err := imu.InitMag()
	if err != nil {
		fmt.Println("left IMU magnetometer disabled:", err)
		return &imuSource{
			imu:      imu,
			magReady: false,
		}, nil
	}

	return &imuSource{
		imu:      imu,
		magCal:   magCal,
		magReady: true,
	}, nil
}

// NewIMUSourceRight initializes the right MPU9250 over SPI.
func NewIMUSourceRight() (IMURawReader, error) {
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

	imu, err := mpu9250.New(tr)

	if err != nil {
		return nil, fmt.Errorf("right IMU new device: %w", err)
	}

	if err := imu.Init(); err != nil {
		return nil, fmt.Errorf("right IMU init: %w", err)
	}

	// Optional self-test & calibration
	if _, err := imu.SelfTest(); err != nil {
		return nil, fmt.Errorf("right IMU self-test: %w", err)
	}
	if err := imu.Calibrate(); err != nil {
		return nil, fmt.Errorf("right IMU calibrate: %w", err)
	}

	// --- Magnetometer init (non-fatal) ---
	magCal, err := imu.InitMag()
	if err != nil {
		fmt.Println("right IMU magnetometer disabled:", err)
		return &imuSource{
			imu:      imu,
			magReady: false,
		}, nil
	}

	return &imuSource{
		imu:      imu,
		magCal:   magCal,
		magReady: true,
	}, nil
}

// IMURawReader reads raw IMU data (accel, gyro, mag).
type IMURawReader interface {
	ReadRaw() (imu_raw.IMURaw, error)
}

// ReadRaw reads accelerometer, gyroscope, and magnetometer data.
func (s *imuSource) ReadRaw() (imu_raw.IMURaw, error) {
	ax, err := s.imu.GetAccelerationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU acc X: %w", err)
	}
	ay, err := s.imu.GetAccelerationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU acc Y: %w", err)
	}
	az, err := s.imu.GetAccelerationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU acc Z: %w", err)
	}

	gx, err := s.imu.GetRotationX()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU gyro X: %w", err)
	}
	gy, err := s.imu.GetRotationY()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU gyro Y: %w", err)
	}
	gz, err := s.imu.GetRotationZ()
	if err != nil {
		return imu_raw.IMURaw{}, fmt.Errorf("IMU gyro Z: %w", err)
	}

	// --- Magnetometer ---
	var mx, my, mz int16
	if s.magReady {
		mag, err := s.imu.ReadMag(s.magCal)
		if err == nil && !mag.Overflow {
			// Store scaled µT values as int16 (simple, consistent)
			mx = int16(mag.X * 10)
			my = int16(mag.Y * 10)
			mz = int16(mag.Z * 10)
		}
	}

	return imu_raw.IMURaw{
		Source: "imu",
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
