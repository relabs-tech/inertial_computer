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

// IntegrateGyro integrates gyroscope data to update pose over time.
// It combines accelerometer-based roll/pitch with gyro-integrated yaw.
//
// Parameters:
//   - ax, ay, az: accelerometer values (for roll/pitch)
//   - gx, gy, gz: gyroscope angular velocities (degrees/second)
//   - prevPose: previous pose state (to integrate yaw from)
//   - deltaTime: elapsed time in seconds since last update
//
// Returns updated Pose with:
//   - Roll, Pitch from accelerometer (complementary filter could be added here)
//   - Yaw integrated from gyroscope Z-axis
func IntegrateGyro(ax, ay, az, gx, gy, gz float64, prevPose Pose, deltaTime float64) Pose {
	// Compute roll and pitch from accelerometer
	pose := ComputePoseFromAccel(ax, ay, az)

	// Integrate gyro Z-axis for yaw
	// yaw_rate is in degrees/second; multiply by deltaTime to get change in degrees
	yawRate := gz // degrees/second
	yawDelta := yawRate * deltaTime
	pose.Yaw = prevPose.Yaw + yawDelta

	// Normalize yaw to [-180, 180]
	for pose.Yaw > 180 {
		pose.Yaw -= 360
	}
	for pose.Yaw < -180 {
		pose.Yaw += 360
	}

	return pose
}

// ComputePoseFromIMURaw computes pose from raw IMU data including gyro integration.
// This is a convenience function that combines accelerometer and gyroscope data.
//
// Parameters:
//   - ax, ay, az: accelerometer values
//   - gx, gy, gz: gyroscope angular velocities (degrees/second)
//   - prevPose: previous pose (for yaw integration)
//   - deltaTime: elapsed time in seconds
//
// Returns integrated Pose.
func ComputePoseFromIMURaw(ax, ay, az, gx, gy, gz float64, prevPose Pose, deltaTime float64) Pose {
	return IntegrateGyro(ax, ay, az, gx, gy, gz, prevPose, deltaTime)
}
