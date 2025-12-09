package sensors

import (
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
)

// TODO: replace with real left IMU + mag read
func ReadLeftIMURaw() (imu_raw.IMURaw, error) {
	return imu_raw.IMURaw{
		Source: "left",
		Ax:     0,
		Ay:     0,
		Az:     0,
		Gx:     0,
		Gy:     0,
		Gz:     0,
		Mx:     0,
		My:     0,
		Mz:     0,
	}, nil
}

// TODO: replace with real right IMU + mag read
func ReadRightIMURaw() (imu_raw.IMURaw, error) {
	return imu_raw.IMURaw{
		Source: "right",
		Ax:     0,
		Ay:     0,
		Az:     0,
		Gx:     0,
		Gy:     0,
		Gz:     0,
		Mx:     0,
		My:     0,
		Mz:     0,
	}, nil
}
