package sensors

import "github.com/relabs-tech/inertial_computer/internal/env"

// ReadLeftEnv reads the LEFT BMP sensor (temp + pressure).
// TODO: replace stub with real BMP driver calls.
func ReadLeftEnv() (env.Sample, error) {
	return env.Sample{
		Source:      "left",
		Temperature: 0,
		Pressure:    0,
	}, nil
}

// ReadRightEnv reads the RIGHT BMP sensor (temp + pressure).
// TODO: replace stub with real BMP driver calls.
func ReadRightEnv() (env.Sample, error) {
	return env.Sample{
		Source:      "right",
		Temperature: 0,
		Pressure:    0,
	}, nil
}
