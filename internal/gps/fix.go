package gps

// Fix represents a single combined GPS fix suitable for JSON and MQTT.
type Fix struct {
	Time       string  `json:"time"`        // e.g. "12:34:56"
	Date       string  `json:"date"`        // e.g. "2025-12-06"
	Latitude   float64 `json:"lat"`         // decimal degrees
	Longitude  float64 `json:"lon"`         // decimal degrees
	SpeedKnots float64 `json:"speed_knots"` // speed over ground
	CourseDeg  float64 `json:"course_deg"`  // course over ground
	Validity   string  `json:"validity"`    // "A" (valid) / "V" (void), etc.
}
