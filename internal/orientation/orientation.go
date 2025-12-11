package orientation

import (
	"math"
)

// Pose is the canonical representation of orientation for your app.
type Pose struct {
	Roll  float64 `json:"roll"`
	Pitch float64 `json:"pitch"`
	Yaw   float64 `json:"yaw"`
}

// Source is anything that can provide poses over time.
// Later you'll have: mock source, IMU source, maybe replay source from file, etc.
type Source interface {
	Next() (Pose, error)
}

// ComputePoseFromAccel computes roll and pitch from accelerometer data only.
// Yaw is set to 0 (placeholder for future magnetometer fusion).
//
// Uses simple tilt formulas:
//
//	roll  = atan2(ay, az)
//	pitch = atan2(-ax, sqrt(ay² + az²))
func ComputePoseFromAccel(ax, ay, az float64) Pose {
	rollRad := math.Atan2(ay, az)
	pitchRad := math.Atan2(-ax, math.Sqrt(ay*ay+az*az))

	rollDeg := rollRad * 180.0 / math.Pi
	pitchDeg := pitchRad * 180.0 / math.Pi

	return Pose{
		Roll:  rollDeg,
		Pitch: pitchDeg,
		Yaw:   0, // placeholder; to be replaced with fused yaw later
	}
}

// AccelToPose computes roll and pitch from raw accelerometer values (in any unit).
// Yaw is set to 0 (placeholder for magnetometer fusion).
// This is a convenience alias for ComputePoseFromAccel.
func AccelToPose(ax, ay, az float64) Pose {
	return ComputePoseFromAccel(ax, ay, az)
}
