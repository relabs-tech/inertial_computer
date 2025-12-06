package orientation

// Pose is the canonical representation of orientation for your app.
// Later this will be filled from the real IMU.
type Pose struct {
	Roll  float64
	Pitch float64
	Yaw   float64
}

// Source is anything that can provide poses over time.
// Later you'll have: mock source, IMU source, maybe replay source from file, etc.
type Source interface {
	Next() (Pose, error)
}
