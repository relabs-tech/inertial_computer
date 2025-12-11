package sensors

import (
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
)

// ReadLeftIMURaw reads raw IMU data from the left MPU9250 sensor.
// Delegates to NewIMUSourceLeft() for real sensor access.
func ReadLeftIMURaw() (imu_raw.IMURaw, error) {
	src, err := NewIMUSourceLeft()
	if err != nil {
		return imu_raw.IMURaw{}, err
	}
	return src.ReadRaw()
}

// ReadRightIMURaw reads raw IMU data from the right MPU9250 sensor.
// Delegates to NewIMUSourceRight() for real sensor access.
func ReadRightIMURaw() (imu_raw.IMURaw, error) {
	src, err := NewIMUSourceRight()
	if err != nil {
		return imu_raw.IMURaw{}, err
	}
	return src.ReadRaw()
}
