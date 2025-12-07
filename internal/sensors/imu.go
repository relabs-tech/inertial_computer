package sensors

import (
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

// TODO: replace with real left IMU + mag read
func ReadLeftIMURaw() (orientation.IMURaw, error) {
	return orientation.IMURaw{
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
func ReadRightIMURaw() (orientation.IMURaw, error) {
	return orientation.IMURaw{
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
