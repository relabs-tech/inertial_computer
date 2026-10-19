// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package gps

// SystemFix is the timing/position snapshot for the *system* GPS (the
// module on /dev/ttyAMA3 + PPS/GPIO6 dedicated to chrony time discipline),
// bridged from gpsd's JSON protocol by the gpsd_producer. This is distinct
// from Fix, which comes from the inertial-computer's own GPS module on
// /dev/serial0 read directly over NMEA — gpsd holds ttyAMA3 exclusively, so
// that device can only be read via gpsd itself, not a second raw NMEA reader.
type SystemFix struct {
	Device  string `json:"device"`
	Time    string `json:"time"`     // ISO8601 UTC, from gpsd TPV
	Mode    int    `json:"mode"`     // 0/1=no fix, 2=2D, 3=3D (gpsd TPV "mode")
	FixType string `json:"fix_type"` // "no fix", "2D", "3D"

	Latitude   float64 `json:"lat"`
	Longitude  float64 `json:"lon"`
	AltitudeM  float64 `json:"altitude_m"`
	SpeedKnots float64 `json:"speed_knots"`
	CourseDeg  float64 `json:"course_deg"`

	HDOP           float64 `json:"hdop"`
	PDOP           float64 `json:"pdop"`
	VDOP           float64 `json:"vdop"`
	SatellitesUsed int     `json:"satellites_used"`
	SatellitesSeen int     `json:"satellites_seen"`

	// Latest PPS pulse vs system-clock offset reported by gpsd (requires the
	// PPS signal wired to GPIO6 and gpsd's pps_gpio/ntpshm support active).
	HavePPS      bool    `json:"have_pps"`
	PPSDevice    string  `json:"pps_device"`
	PPSOffsetSec float64 `json:"pps_offset_sec"`
}
