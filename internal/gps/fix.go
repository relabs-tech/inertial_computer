// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package gps

// Satellite represents information about a single GPS satellite.
type Satellite struct {
	SVNumber  int64 `json:"sv_number"` // satellite vehicle number (PRN)
	Elevation int64 `json:"elevation"` // elevation in degrees (0-90)
	Azimuth   int64 `json:"azimuth"`   // azimuth in degrees (0-359)
	SNR       int64 `json:"snr"`       // signal-to-noise ratio in dB (0-99, 0=not tracked)
}

// Position contains GPS position and timing data (from RMC and GGA).
type Position struct {
	Time      string  `json:"time"`       // e.g. "12:34:56"
	Date      string  `json:"date"`       // e.g. "2025-12-06"
	Latitude  float64 `json:"lat"`        // decimal degrees
	Longitude float64 `json:"lon"`        // decimal degrees
	Altitude  float64 `json:"altitude_m"` // altitude above mean sea level (meters)
	Validity  string  `json:"validity"`   // "A" (valid) / "V" (void)
}

// Velocity contains speed and course data (from RMC and VTG).
type Velocity struct {
	SpeedKnots float64 `json:"speed_knots"` // speed over ground (knots)
	SpeedKmh   float64 `json:"speed_kmh"`   // speed over ground (km/h)
	CourseDeg  float64 `json:"course_deg"`  // course over ground (degrees)
}

// Quality contains fix quality and DOP metrics (from GGA and GSA).
type Quality struct {
	FixType       string  `json:"fix_type"`       // "2D", "3D", or "no fix"
	FixQuality    string  `json:"fix_quality"`    // invalid/GPS/DGPS/RTK
	NumSatellites int64   `json:"num_satellites"` // number of satellites in use
	HDOP          float64 `json:"hdop"`           // horizontal dilution of precision
	PDOP          float64 `json:"pdop"`           // position dilution of precision
	VDOP          float64 `json:"vdop"`           // vertical dilution of precision
}

// SatellitesInView contains all visible satellites with signal strength (from GSV).
type SatellitesInView struct {
	GPSSatellites     []Satellite `json:"gps_satellites"`     // GPS satellites (from GPGSV)
	GLONASSSatellites []Satellite `json:"glonass_satellites"` // GLONASS satellites (from GLGSV)
	GPSCount          int         `json:"gps_count"`          // GPS satellite count
	GLONASSCount      int         `json:"glonass_count"`      // GLONASS satellite count
}

// Fix represents a single combined GPS fix (for backwards compatibility or full data).
// Data is accumulated from multiple NMEA sentence types (RMC, GGA, GSA, VTG, GSV).
type Fix struct {
	// From RMC (Recommended Minimum)
	Time       string  `json:"time"`        // e.g. "12:34:56"
	Date       string  `json:"date"`        // e.g. "2025-12-06"
	Latitude   float64 `json:"lat"`         // decimal degrees
	Longitude  float64 `json:"lon"`         // decimal degrees
	SpeedKnots float64 `json:"speed_knots"` // speed over ground (knots)
	CourseDeg  float64 `json:"course_deg"`  // course over ground (degrees)
	Validity   string  `json:"validity"`    // "A" (valid) / "V" (void)

	// From GGA (Global Positioning System Fix Data)
	Altitude      float64 `json:"altitude_m"`     // altitude above mean sea level (meters)
	FixQuality    string  `json:"fix_quality"`    // 0=invalid, 1=GPS, 2=DGPS, etc.
	NumSatellites int64   `json:"num_satellites"` // number of satellites in use
	HDOP          float64 `json:"hdop"`           // horizontal dilution of precision

	// From GSA (GPS DOP and Active Satellites)
	FixType string  `json:"fix_type"` // "2D", "3D", or "no fix"
	PDOP    float64 `json:"pdop"`     // position dilution of precision
	VDOP    float64 `json:"vdop"`     // vertical dilution of precision

	// From VTG (Track Made Good and Ground Speed)
	SpeedKmh float64 `json:"speed_kmh"` // speed over ground (km/h)

	// From GSV (GPS Satellites in View)
	GPSSatellitesInView     []Satellite `json:"gps_satellites_in_view"`     // GPS satellites with signal strength
	GLONASSSatellitesInView []Satellite `json:"glonass_satellites_in_view"` // GLONASS satellites with signal strength
}
