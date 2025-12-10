package imu

// IMURaw represents a single raw IMU+mag sample.
type IMURaw struct {
	Source string `json:"source"` // "left" or "right"

	Ax int16 `json:"ax"` // accel
	Ay int16 `json:"ay"`
	Az int16 `json:"az"`

	Gx int16 `json:"gx"` // gyro
	Gy int16 `json:"gy"`
	Gz int16 `json:"gz"`

	Mx int16 `json:"mx"` // magnetometer
	My int16 `json:"my"`
	Mz int16 `json:"mz"`
}

type IMURawSource interface {
	NextRaw() (IMURaw, error)
}
