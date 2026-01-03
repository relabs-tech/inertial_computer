// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


package app

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// CalibrationSession holds the state of an active calibration
type CalibrationSession struct {
	IMU          string
	Conn         *websocket.Conn
	mu           sync.Mutex
	currentPhase string
	currentStep  int
	results      CalibrationResult
}

// CalibrationResult matches the structure from cmd/calibration/main.go
type CalibrationResult struct {
	Version   int       `json:"version"`
	IMU       string    `json:"imu"`
	Timestamp time.Time `json:"timestamp"`

	// Gyroscope calibration
	GyroBiasX         float64 `json:"gyro_bias_x"`
	GyroBiasY         float64 `json:"gyro_bias_y"`
	GyroBiasZ         float64 `json:"gyro_bias_z"`
	GyroConfidence    float64 `json:"gyro_confidence"`
	GyroStaticStdDev  float64 `json:"gyro_static_stddev"`
	GyroDynamicStdDev float64 `json:"gyro_dynamic_stddev"`

	// Accelerometer calibration
	AccelBiasX      float64 `json:"accel_bias_x"`
	AccelBiasY      float64 `json:"accel_bias_y"`
	AccelBiasZ      float64 `json:"accel_bias_z"`
	AccelScaleX     float64 `json:"accel_scale_x"`
	AccelScaleY     float64 `json:"accel_scale_y"`
	AccelScaleZ     float64 `json:"accel_scale_z"`
	AccelConfidence float64 `json:"accel_confidence"`
	AccelAvgStdDev  float64 `json:"accel_avg_stddev"`

	// Magnetometer calibration
	MagOffsetX     float64 `json:"mag_offset_x"`
	MagOffsetY     float64 `json:"mag_offset_y"`
	MagOffsetZ     float64 `json:"mag_offset_z"`
	MagScaleX      float64 `json:"mag_scale_x"`
	MagScaleY      float64 `json:"mag_scale_y"`
	MagScaleZ      float64 `json:"mag_scale_z"`
	MagConfidence  float64 `json:"mag_confidence"`
	MagRangeX      float64 `json:"mag_range_x"`
	MagRangeY      float64 `json:"mag_range_y"`
	MagRangeZ      float64 `json:"mag_range_z"`
	MagSampleCount int     `json:"mag_sample_count"`

	TotalSamples int `json:"total_samples"`
}

// WebSocket message types
type WSMessage struct {
	Action string `json:"action"` // init, next, cancel
	IMU    string `json:"imu,omitempty"`
}

type WSResponse struct {
	Type     string                 `json:"type"` // phase, step, progress, stats, complete, error
	Phase    string                 `json:"phase,omitempty"`
	Step     string                 `json:"step,omitempty"`
	Progress float64                `json:"progress,omitempty"`
	Stats    map[string]interface{} `json:"stats,omitempty"`
	Results  interface{}            `json:"results,omitempty"`
	Message  string                 `json:"message,omitempty"`
}

// HandleCalibrationWS handles the WebSocket connection for calibration
func HandleCalibrationWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("calibration: websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	session := &CalibrationSession{
		Conn: conn,
		results: CalibrationResult{
			Version:     1,
			Timestamp:   time.Now(),
			AccelScaleX: 1.0,
			AccelScaleY: 1.0,
			AccelScaleZ: 1.0,
			MagScaleX:   1.0,
			MagScaleY:   1.0,
			MagScaleZ:   1.0,
		},
	}

	// Main message loop
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("calibration: websocket read error: %v", err)
			break
		}

		switch msg.Action {
		case "init":
			session.IMU = msg.IMU
			session.results.IMU = msg.IMU
			log.Printf("calibration: initialized for IMU: %s", msg.IMU)

		case "next":
			session.mu.Lock()
			err := session.runNextStep()
			session.mu.Unlock()
			if err != nil {
				session.sendError(err.Error())
			}

		case "cancel":
			log.Printf("calibration: cancelled by user")
			return
		}
	}
}

func (s *CalibrationSession) runNextStep() error {
	// State machine for calibration phases
	switch s.currentPhase {
	case "":
		// Start with gyroscope
		s.currentPhase = "gyro"
		s.currentStep = 0
		return s.runGyroStep()

	case "gyro":
		s.currentStep++
		if s.currentStep >= 4 {
			// Move to accelerometer
			s.currentPhase = "accel"
			s.currentStep = 0
			return s.runAccelStep()
		}
		return s.runGyroStep()

	case "accel":
		s.currentStep++
		if s.currentStep >= 6 {
			// Move to magnetometer
			s.currentPhase = "mag"
			s.currentStep = 0
			return s.runMagStep()
		}
		return s.runAccelStep()

	case "mag":
		// Complete calibration
		return s.complete()
	}

	return nil
}

func (s *CalibrationSession) runGyroStep() error {
	s.sendPhase("gyro")

	mgr := sensors.GetIMUManager()
	if mgr == nil {
		return fmt.Errorf("IMU manager not initialized")
	}

	// Check which IMU is available
	var readFunc func() (imu_raw.IMURaw, error)
	if s.IMU == "left" {
		if !mgr.IsLeftIMUAvailable() {
			return fmt.Errorf("left IMU not available")
		}
		readFunc = mgr.ReadLeftIMU
	} else {
		if !mgr.IsRightIMUAvailable() {
			return fmt.Errorf("right IMU not available")
		}
		readFunc = mgr.ReadRightIMU
	}

	steps := []string{"gyro-static", "gyro-x", "gyro-y", "gyro-z"}
	stepID := steps[s.currentStep]
	s.sendStep(stepID, "gyro")

	switch s.currentStep {
	case 0: // Static calibration
		s.sendProgress(5)
		time.Sleep(1 * time.Second) // Give user time to place device

		samples := make([][3]float64, 0, 100)
		for i := 0; i < 100; i++ {
			reading, err := readFunc()
			if err != nil {
				return err
			}
			samples = append(samples, [3]float64{
				float64(reading.Gx),
				float64(reading.Gy),
				float64(reading.Gz),
			})
			s.sendProgress(5 + float64(i)*0.9)
			time.Sleep(100 * time.Millisecond)
		}

		// Calculate bias
		s.results.GyroBiasX = mean(samples, 0)
		s.results.GyroBiasY = mean(samples, 1)
		s.results.GyroBiasZ = mean(samples, 2)
		s.results.GyroStaticStdDev = (stddev(samples, 0) + stddev(samples, 1) + stddev(samples, 2)) / 3.0
		s.results.TotalSamples += len(samples)

	default: // Dynamic rotation steps
		s.sendProgress(float64(s.currentStep) * 25)
		time.Sleep(1 * time.Second)

		samples := make([][3]float64, 0, 50)
		for i := 0; i < 50; i++ {
			reading, err := readFunc()
			if err != nil {
				return err
			}
			// Apply current bias correction
			corrected := [3]float64{
				float64(reading.Gx) - s.results.GyroBiasX,
				float64(reading.Gy) - s.results.GyroBiasY,
				float64(reading.Gz) - s.results.GyroBiasZ,
			}
			samples = append(samples, corrected)
			s.sendProgress(float64(s.currentStep)*25 + float64(i)*0.5)
			time.Sleep(100 * time.Millisecond)
		}

		// Calculate dynamic standard deviation
		dynamicStdDev := (stddev(samples, 0) + stddev(samples, 1) + stddev(samples, 2)) / 3.0
		if s.currentStep == 1 {
			s.results.GyroDynamicStdDev = dynamicStdDev
		} else {
			s.results.GyroDynamicStdDev = (s.results.GyroDynamicStdDev + dynamicStdDev) / 2.0
		}
		s.results.TotalSamples += len(samples)
	}

	// Calculate confidence
	if s.results.GyroStaticStdDev > 0 {
		s.results.GyroConfidence = 100.0 / (1.0 + s.results.GyroStaticStdDev*1000.0)
	}

	s.sendStats()
	s.sendActionReady()
	return nil
}

func (s *CalibrationSession) runAccelStep() error {
	s.sendPhase("accel")

	mgr := sensors.GetIMUManager()
	if mgr == nil {
		return fmt.Errorf("IMU manager not initialized")
	}

	// Check which IMU is available
	var readFunc func() (imu_raw.IMURaw, error)
	if s.IMU == "left" {
		if !mgr.IsLeftIMUAvailable() {
			return fmt.Errorf("left IMU not available")
		}
		readFunc = mgr.ReadLeftIMU
	} else {
		if !mgr.IsRightIMUAvailable() {
			return fmt.Errorf("right IMU not available")
		}
		readFunc = mgr.ReadRightIMU
	}

	steps := []string{"accel-up", "accel-down", "accel-right", "accel-left", "accel-forward", "accel-back"}
	stepID := steps[s.currentStep]
	s.sendStep(stepID, "accel")
	s.sendProgress(float64(s.currentStep) * 16.67)

	time.Sleep(2 * time.Second) // Give user time to position device

	// Collect samples for this orientation
	samples := make([][3]float64, 0, 50)
	for i := 0; i < 50; i++ {
		reading, err := readFunc()
		if err != nil {
			return err
		}
		samples = append(samples, [3]float64{
			float64(reading.Ax),
			float64(reading.Ay),
			float64(reading.Az),
		})
		s.sendProgress(float64(s.currentStep)*16.67 + float64(i)*0.33)
		time.Sleep(100 * time.Millisecond)
	}

	// Calculate mean for this orientation
	meanX := mean(samples, 0)
	meanY := mean(samples, 1)
	meanZ := mean(samples, 2)

	// Expected gravity values for each orientation (in g's)
	expected := [][3]float64{
		{0, 0, 1},  // up
		{0, 0, -1}, // down
		{1, 0, 0},  // right
		{-1, 0, 0}, // left
		{0, 1, 0},  // forward
		{0, -1, 0}, // back
	}
	_ = expected // Mark as used

	// Accumulate for bias and scale calculation
	// Simple approach: use opposing pairs to calculate bias and scale
	switch s.currentStep {
	case 0: // Z+ up
		s.results.AccelScaleZ = 1.0 / meanZ
	case 1: // Z- down
		s.results.AccelBiasZ = (meanZ/s.results.AccelScaleZ + 1.0) / 2.0
	case 2: // X+ right
		s.results.AccelScaleX = 1.0 / meanX
	case 3: // X- left
		s.results.AccelBiasX = (meanX/s.results.AccelScaleX + 1.0) / 2.0
	case 4: // Y+ forward
		s.results.AccelScaleY = 1.0 / meanY
	case 5: // Y- back
		s.results.AccelBiasY = (meanY/s.results.AccelScaleY + 1.0) / 2.0
	}

	s.results.TotalSamples += len(samples)

	// Calculate standard deviation for this orientation
	avgStdDev := (stddev(samples, 0) + stddev(samples, 1) + stddev(samples, 2)) / 3.0
	if s.currentStep == 0 {
		s.results.AccelAvgStdDev = avgStdDev
	} else {
		s.results.AccelAvgStdDev = (s.results.AccelAvgStdDev*float64(s.currentStep) + avgStdDev) / float64(s.currentStep+1)
	}

	// Calculate confidence
	if s.results.AccelAvgStdDev > 0 {
		s.results.AccelConfidence = 100.0 / (1.0 + s.results.AccelAvgStdDev*100.0)
	}

	s.sendStats()
	s.sendActionReady()
	return nil
}

func (s *CalibrationSession) runMagStep() error {
	s.sendPhase("mag")
	s.sendStep("mag-calibrate", "mag")
	s.sendProgress(0)

	mgr := sensors.GetIMUManager()
	if mgr == nil {
		return fmt.Errorf("IMU manager not initialized")
	}

	// Check which IMU is available
	var readFunc func() (imu_raw.IMURaw, error)
	if s.IMU == "left" {
		if !mgr.IsLeftIMUAvailable() {
			return fmt.Errorf("left IMU not available")
		}
		readFunc = mgr.ReadLeftIMU
	} else {
		if !mgr.IsRightIMUAvailable() {
			return fmt.Errorf("right IMU not available")
		}
		readFunc = mgr.ReadRightIMU
	}

	time.Sleep(2 * time.Second) // Give user time to start moving

	// Collect magnetometer samples for 20 seconds
	samples := make([][3]float64, 0, 200)
	minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	for i := 0; i < 200; i++ {
		reading, err := readFunc()
		if err != nil {
			return err
		}

		mx, my, mz := float64(reading.Mx), float64(reading.My), float64(reading.Mz)
		samples = append(samples, [3]float64{mx, my, mz})

		// Track min/max for each axis
		if mx < minX {
			minX = mx
		}
		if mx > maxX {
			maxX = mx
		}
		if my < minY {
			minY = my
		}
		if my > maxY {
			maxY = my
		}
		if mz < minZ {
			minZ = mz
		}
		if mz > maxZ {
			maxZ = mz
		}

		s.sendProgress(float64(i) * 0.5)
		time.Sleep(100 * time.Millisecond)
	}

	// Calculate hard-iron offsets (center of ellipsoid)
	s.results.MagOffsetX = (maxX + minX) / 2.0
	s.results.MagOffsetY = (maxY + minY) / 2.0
	s.results.MagOffsetZ = (maxZ + minZ) / 2.0

	// Calculate soft-iron scale factors (diagonal approximation)
	rangeX := maxX - minX
	rangeY := maxY - minY
	rangeZ := maxZ - minZ
	avgRange := (rangeX + rangeY + rangeZ) / 3.0

	s.results.MagScaleX = avgRange / rangeX
	s.results.MagScaleY = avgRange / rangeY
	s.results.MagScaleZ = avgRange / rangeZ

	s.results.MagRangeX = rangeX
	s.results.MagRangeY = rangeY
	s.results.MagRangeZ = rangeZ
	s.results.MagSampleCount = len(samples)
	s.results.TotalSamples += len(samples)

	// Calculate confidence based on range coverage
	minRange := math.Min(rangeX, math.Min(rangeY, rangeZ))
	maxRange := math.Max(rangeX, math.Max(rangeY, rangeZ))
	rangeRatio := minRange / maxRange
	s.results.MagConfidence = rangeRatio * 100.0

	s.sendProgress(100)
	s.sendStats()

	// Don't send action ready - auto-proceed to complete
	time.Sleep(1 * time.Second)
	return s.complete()
}

func (s *CalibrationSession) complete() error {
	// Save results to file
	filename := fmt.Sprintf("%s_%d_inertial_calibration.json", s.IMU, time.Now().Unix())

	// Use current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	filepath := filepath.Join(cwd, filename)

	data, err := json.MarshalIndent(s.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal calibration results: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write calibration file: %w", err)
	}

	log.Printf("calibration: saved results to %s", filepath)

	// Send completion message
	s.Conn.WriteJSON(WSResponse{
		Type:    "complete",
		Results: map[string]interface{}{"filename": filename},
	})

	return nil
}

func (s *CalibrationSession) sendPhase(phase string) {
	s.Conn.WriteJSON(WSResponse{
		Type:  "phase",
		Phase: phase,
	})
}

func (s *CalibrationSession) sendStep(step, phase string) {
	s.Conn.WriteJSON(WSResponse{
		Type:  "step",
		Step:  step,
		Phase: phase,
	})
}

func (s *CalibrationSession) sendProgress(progress float64) {
	s.Conn.WriteJSON(WSResponse{
		Type:     "progress",
		Progress: progress,
	})
}

func (s *CalibrationSession) sendStats() {
	stats := map[string]interface{}{
		"gyro":    s.results.GyroConfidence,
		"accel":   s.results.AccelConfidence,
		"mag":     s.results.MagConfidence,
		"samples": s.results.TotalSamples,
	}
	s.Conn.WriteJSON(WSResponse{
		Type:  "stats",
		Stats: stats,
	})
}

func (s *CalibrationSession) sendActionReady() {
	s.Conn.WriteJSON(WSResponse{
		Type:    "action",
		Message: "ready",
	})
}

func (s *CalibrationSession) sendError(message string) {
	s.Conn.WriteJSON(WSResponse{
		Type:    "error",
		Message: message,
	})
}

// Helper functions for statistics
func mean(data [][3]float64, axis int) float64 {
	sum := 0.0
	for _, v := range data {
		sum += v[axis]
	}
	return sum / float64(len(data))
}

func stddev(data [][3]float64, axis int) float64 {
	if len(data) == 0 {
		return 0
	}
	m := mean(data, axis)
	variance := 0.0
	for _, v := range data {
		diff := v[axis] - m
		variance += diff * diff
	}
	variance /= float64(len(data))
	return math.Sqrt(variance)
}
