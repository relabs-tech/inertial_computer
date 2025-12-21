package env

// Sample represents a single environmental measurement (BMP).
type Sample struct {
	Source string `json:"source"` // "left" or "right"`

	Temperature  float64 `json:"temp_c"`        // °C
	Pressure     float64 `json:"pressure_pa"`   // Pa
	PressureMbar float64 `json:"pressure_mbar"` // mbar
	PressureHPa  float64 `json:"pressure_hpa"`  // hPa
}

type EnvSource interface {
	NextEnv() (Sample, error)
}
