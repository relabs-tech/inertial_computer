// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text


// ./cmd/calibration/main.go
//
// Guided calibration for MPU-9250 class IMUs in this project.
// Calibrates:
//  1. Gyro: static bias (still) + dynamic refinement via guided rotations (X/Y/Z)
//  2. Accel: 6-point (±X, ±Y, ±Z) static poses to estimate bias + per-axis scale
//  3. Mag: guided 3D rotation to estimate hard-iron offset + per-axis soft-iron scale (min/max method)
//
// Output:
//
//	Writes a JSON file under ./calibration/ including calibration date/time and quality/confidence.
//
// Run:
//
//	go run ./cmd/calibration
//
// Notes / assumptions:
//   - Reads raw samples via internal/sensors IMUManager (left/right) returning internal/imu.IMURaw.
//   - Stores calibration in RAW UNITS (counts). Applying this calibration later requires consistent units.
//   - Mag calibration here uses a practical min/max ellipsoid approximation (offset + diagonal scale). It is
//     robust and easy, though not as accurate as a full 3x3 ellipsoid fit.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

const (
	sampleHz = 100 // target loop frequency (best-effort)

	// Gyro
	gyroStaticDuration = 10 * time.Second
	gyroRotMinDur      = 8 * time.Second
	gyroRotMaxDur      = 30 * time.Second

	// Accel 6-point
	accelPoseDuration = 6 * time.Second

	// Mag
	magDurationDefault = 60 * time.Second

	// Generic quality heuristics (in raw counts; tune as needed)
	stillStdGood = 3.0  // "good" standard deviation threshold for stillness
	stillStdBad  = 12.0 // "bad" threshold; above this confidence drops steeply

	dominanceGood = 0.70 // dominant-axis ratio for guided single-axis rotations
	dominanceBad  = 0.45

	minMeanAbsRate = 20.0 // minimal mean abs gyro rate (counts) to consider "real rotation"

	// Confidence floor (we never want hard zero unless we error out)
	confFloor = 0.05
)

// ---------- Data model (JSON output) ----------

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type PhaseStats struct {
	Samples       int      `json:"samples"`
	DurationSec   float64  `json:"duration_sec"`
	Mean          Vec3     `json:"mean"`
	MeanAbs       Vec3     `json:"mean_abs"`
	StdDev        Vec3     `json:"stddev"`
	AxisDominance Vec3     `json:"axis_dominance,omitempty"`
	Integrated    Vec3     `json:"integrated,omitempty"` // ∫(value) dt in (counts*sec) for gyro rotations
	Notes         []string `json:"notes,omitempty"`
}

type AccelPoseStats struct {
	Pose        string  `json:"pose"`
	Samples     int     `json:"samples"`
	DurationSec float64 `json:"duration_sec"`
	Mean        Vec3    `json:"mean"`
	StdDev      Vec3    `json:"stddev"`
	Confidence  float64 `json:"confidence"`
}

type CalibrationResult struct {
	SchemaVersion int    `json:"schema_version"`
	CalibrationAt string `json:"calibration_at"` // RFC3339
	IMU           string `json:"imu"`            // "left" or "right"

	// Gyro bias (counts)
	GyroBiasStatic Vec3 `json:"gyro_bias_static"`
	GyroBiasDyn    Vec3 `json:"gyro_bias_dynamic"`
	GyroBiasFinal  Vec3 `json:"gyro_bias_final"`

	// Accel bias + scale (counts)
	// CorrectedAccelAxis = (raw - bias) / scale
	AccelBias  Vec3 `json:"accel_bias"`
	AccelScale Vec3 `json:"accel_scale"`

	// Mag hard/soft iron approximation (counts)
	// CorrectedMagAxis = (raw - offset) / scale
	MagOffset Vec3 `json:"mag_offset"`
	MagScale  Vec3 `json:"mag_scale"`

	// Confidence components and overall
	Confidence struct {
		GyroStatic float64 `json:"gyro_static"`
		GyroRot    float64 `json:"gyro_rotation"`
		Accel6Pt   float64 `json:"accel_6pt"`
		Mag        float64 `json:"mag"`
		Overall    float64 `json:"overall"`
	} `json:"confidence"`

	// Supporting stats
	GyroStaticStats PhaseStats            `json:"gyro_static_stats"`
	GyroRotStats    map[string]PhaseStats `json:"gyro_rotation_stats"` // keys: "x", "y", "z"

	AccelPoseStats []AccelPoseStats `json:"accel_pose_stats"`

	MagStats PhaseStats `json:"mag_stats"`

	Notes []string `json:"notes,omitempty"`
}

// ---------- Main ----------

func main() {
	in := bufio.NewReader(os.Stdin)

	// Parse command-line flags
	configPath := flag.String("config", "inertial_config.txt", "Path to configuration file")
	flag.Parse()

	fmt.Println("=== Guided Calibration (Accel + Gyro + Mag) ===")
	fmt.Println("This workflow will prompt you in the console and store results in ./inertial_calibration.json")
	fmt.Println()

	// Initialize configuration
	if err := config.InitGlobal(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load config from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	// Init IMUs
	mgr := sensors.GetIMUManager()
	if err := mgr.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: IMU init failed: %v\n", err)
		os.Exit(1)
	}

	leftOK := mgr.IsLeftIMUAvailable()
	rightOK := mgr.IsRightIMUAvailable()
	if !leftOK && !rightOK {
		fmt.Fprintln(os.Stderr, "ERROR: No IMU available (left and right both unavailable).")
		os.Exit(1)
	}

	imuName, readFn := pickIMU(in, leftOK, rightOK, mgr)

	fmt.Printf("\nSelected IMU: %s\n\n", imuName)

	res := CalibrationResult{
		SchemaVersion: 1,
		CalibrationAt: time.Now().Format(time.RFC3339),
		IMU:           imuName,
		GyroRotStats:  map[string]PhaseStats{},
	}

	// ---------------- Gyro calibration ----------------
	fmt.Println("Step 1/3 — Gyro static bias")
	fmt.Println("Place the device on a stable surface and do not touch it.")
	waitEnter(in, "Press ENTER to start static gyro bias capture (10s)...")

	gyroStaticSamples, sStats, err := captureSamples(readFn, gyroStaticDuration, func(r imu.IMURaw) Vec3 {
		return Vec3{X: float64(r.Gx), Y: float64(r.Gy), Z: float64(r.Gz)}
	})
	if err != nil {
		fatal(err)
	}
	res.GyroStaticStats = sStats
	res.GyroBiasStatic = sStats.Mean

	gyroStaticConf := stillnessConfidence(sStats.StdDev)
	res.Confidence.GyroStatic = gyroStaticConf

	fmt.Printf("Static gyro bias (counts): X=%.2f Y=%.2f Z=%.2f | confidence=%.2f\n",
		res.GyroBiasStatic.X, res.GyroBiasStatic.Y, res.GyroBiasStatic.Z, gyroStaticConf)

	// Gyro dynamic refinement
	fmt.Println("\nStep 1b/3 — Gyro dynamic refinement via guided rotations")
	fmt.Println("For each axis (X, Y, Z), rotate the device 2–3 full turns around that axis.")
	fmt.Println("Try to keep the rotation mostly around the prompted axis.")
	fmt.Println("You will press ENTER to start capture and ENTER again to stop (or it stops automatically).")
	fmt.Println()

	gyroDynBias, gyroRotConf := guidedGyroRotations(in, readFn, res.GyroBiasStatic, &res)
	res.GyroBiasDyn = gyroDynBias

	// Combine static and dynamic (favor static but incorporate motion-validated bias)
	alpha := 0.75
	res.GyroBiasFinal = Vec3{
		X: alpha*res.GyroBiasStatic.X + (1-alpha)*res.GyroBiasDyn.X,
		Y: alpha*res.GyroBiasStatic.Y + (1-alpha)*res.GyroBiasDyn.Y,
		Z: alpha*res.GyroBiasStatic.Z + (1-alpha)*res.GyroBiasDyn.Z,
	}
	res.Confidence.GyroRot = gyroRotConf

	fmt.Printf("Dynamic gyro bias (counts): X=%.2f Y=%.2f Z=%.2f | confidence=%.2f\n",
		res.GyroBiasDyn.X, res.GyroBiasDyn.Y, res.GyroBiasDyn.Z, gyroRotConf)
	fmt.Printf("Final gyro bias (counts):   X=%.2f Y=%.2f Z=%.2f\n",
		res.GyroBiasFinal.X, res.GyroBiasFinal.Y, res.GyroBiasFinal.Z)

	_ = gyroStaticSamples // kept for possible future extensions

	// ---------------- Accel calibration (6-point) ----------------
	fmt.Println("\nStep 2/3 — Accelerometer 6-point calibration (bias + scale)")
	fmt.Println("You will place the device still in 6 orientations: +X, -X, +Y, -Y, +Z, -Z (axis UP).")
	fmt.Println("Each pose captures 6 seconds. Keep it as still as possible.")
	fmt.Println()

	accBias, accScale, accConf, poseStats, err := guidedAccel6Point(in, readFn)
	if err != nil {
		fatal(err)
	}
	res.AccelBias = accBias
	res.AccelScale = accScale
	res.Confidence.Accel6Pt = accConf
	res.AccelPoseStats = poseStats

	fmt.Printf("Accel bias (counts):  X=%.2f Y=%.2f Z=%.2f\n", accBias.X, accBias.Y, accBias.Z)
	fmt.Printf("Accel scale (counts): X=%.2f Y=%.2f Z=%.2f | confidence=%.2f\n", accScale.X, accScale.Y, accScale.Z, accConf)

	// ---------------- Mag calibration ----------------
	fmt.Println("\nStep 3/3 — Magnetometer calibration (offset + diagonal scale)")
	fmt.Println("Rotate the device through all orientations (3D).")
	fmt.Println("Move away from large metal objects and power cables if possible.")
	fmt.Println("You can stop early by pressing ENTER again.")
	fmt.Println()

	waitEnter(in, "Press ENTER to start magnetometer capture (default 60s, ENTER to stop earlier)...")

	magOffset, magScale, magConf, magStats, err := guidedMag(in, readFn, magDurationDefault)
	if err != nil {
		fatal(err)
	}
	res.MagOffset = magOffset
	res.MagScale = magScale
	res.Confidence.Mag = magConf
	res.MagStats = magStats

	fmt.Printf("Mag offset (counts): X=%.2f Y=%.2f Z=%.2f\n", magOffset.X, magOffset.Y, magOffset.Z)
	fmt.Printf("Mag scale (counts):  X=%.2f Y=%.2f Z=%.2f | confidence=%.2f\n",
		magScale.X, magScale.Y, magScale.Z, magConf)

	// ---------------- Overall confidence + store ----------------
	res.Confidence.Overall = overallConfidence(res.Confidence.GyroStatic, res.Confidence.GyroRot, res.Confidence.Accel6Pt, res.Confidence.Mag)

	if err := writeResult(res); err != nil {
		fatal(err)
	}

	fmt.Println("\nCalibration complete.")
	fmt.Printf("Overall confidence: %.2f\n", res.Confidence.Overall)
	fmt.Println("Saved to ./inertial_calibration.json")
}

// ---------- IMU selection ----------

func pickIMU(in *bufio.Reader, leftOK, rightOK bool, mgr *sensors.IMUManager) (string, func() (imu.IMURaw, error)) {
	if leftOK && !rightOK {
		fmt.Println("Only left IMU available, using left IMU.")
		time.Sleep(5 * time.Second)
		return "left", func() (imu.IMURaw, error) { return mgr.ReadLeftIMU() }
	}
	if rightOK && !leftOK {
		fmt.Println("Only right IMU available, using right IMU.")
		time.Sleep(5 * time.Second)
		return "right", func() (imu.IMURaw, error) { return mgr.ReadRightIMU() }
	}

	fmt.Println()
	fmt.Println("Both IMUs available.")
	time.Sleep(5 * time.Second) // Give user time to see the message
	for {
		fmt.Print("Select IMU to calibrate [L/R] (default: L): ")
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(strings.ToUpper(line))
		if line == "" || line == "L" {
			return "left", func() (imu.IMURaw, error) { return mgr.ReadLeftIMU() }
		}
		if line == "R" {
			return "right", func() (imu.IMURaw, error) { return mgr.ReadRightIMU() }
		}
		fmt.Println("Invalid input. Type 'L' or 'R'.")
	}
}

// ---------- Guided gyro rotations ----------

func guidedGyroRotations(in *bufio.Reader, readFn func() (imu.IMURaw, error), bStatic Vec3, res *CalibrationResult) (Vec3, float64) {
	type axisResult struct {
		axis string
		bias float64
		conf float64
	}
	results := []axisResult{}

	for _, axis := range []string{"x", "y", "z"} {
		fmt.Printf("Axis %s rotation: rotate mostly around %s-axis (2–3 full turns).\n", strings.ToUpper(axis), strings.ToUpper(axis))
		waitEnter(in, "Press ENTER to start capture, then ENTER again to stop...")

		rotSamples, stats, err := captureUntilEnterOrTimeout(in, readFn, gyroRotMaxDur, func(r imu.IMURaw) Vec3 {
			// subtract static bias before integrating & stats
			return Vec3{
				X: float64(r.Gx) - bStatic.X,
				Y: float64(r.Gy) - bStatic.Y,
				Z: float64(r.Gz) - bStatic.Z,
			}
		})
		if err != nil {
			fmt.Printf("Warning: rotation capture failed for axis %s: %v\n", axis, err)
			stats.Notes = append(stats.Notes, "capture_error: "+err.Error())
			res.GyroRotStats[axis] = stats
			results = append(results, axisResult{axis: axis, bias: 0, conf: confFloor})
			continue
		}

		// Enforce minimum duration
		if stats.DurationSec < gyroRotMinDur.Seconds() {
			stats.Notes = append(stats.Notes, fmt.Sprintf("too_short: %.2fs < %.2fs", stats.DurationSec, gyroRotMinDur.Seconds()))
		}

		// Compute per-axis dominance and integrated angle proxy
		intg := integrate(rotSamples)
		stats.Integrated = intg
		stats.AxisDominance = axisDominance(stats.MeanAbs)

		// Residual bias estimate: b = ∫ω dt / T (counts)
		var b float64
		switch axis {
		case "x":
			b = intg.X / stats.DurationSec
		case "y":
			b = intg.Y / stats.DurationSec
		case "z":
			b = intg.Z / stats.DurationSec
		}

		// Confidence heuristic for this axis
		conf := rotationConfidence(axis, stats)

		res.GyroRotStats[axis] = stats
		results = append(results, axisResult{axis: axis, bias: b, conf: conf})

		fmt.Printf("  Axis %s: residual bias=%.2f counts | dominance=%.2f | meanAbs=%.2f | conf=%.2f\n",
			strings.ToUpper(axis), b, dominantForAxis(axis, stats.AxisDominance), meanAbsForAxis(axis, stats.MeanAbs), conf)
	}

	// Combine axis biases
	bDyn := Vec3{}
	conf := 0.0
	weights := 0.0

	for _, r := range results {
		w := clamp01(r.conf)
		weights += w
		conf += w * r.conf
		switch r.axis {
		case "x":
			bDyn.X = r.bias
		case "y":
			bDyn.Y = r.bias
		case "z":
			bDyn.Z = r.bias
		}
	}
	if weights > 0 {
		conf = conf / weights
	} else {
		conf = confFloor
	}
	return bDyn, clamp01(conf)
}

// ---------- Guided accel 6-point ----------

func guidedAccel6Point(in *bufio.Reader, readFn func() (imu.IMURaw, error)) (bias Vec3, scale Vec3, confidence float64, poseStats []AccelPoseStats, err error) {
	poses := []string{"+X", "-X", "+Y", "-Y", "+Z", "-Z"}

	type poseData struct {
		pose string
		mean Vec3
		std  Vec3
		conf float64
	}
	data := map[string]poseData{}

	for _, p := range poses {
		fmt.Printf("Pose %s UP: place the device so %s axis points upward, then keep it still.\n", p, p)
		waitEnter(in, "Press ENTER to start capture (6s)...")

		_, stats, e := captureSamples(readFn, accelPoseDuration, func(r imu.IMURaw) Vec3 {
			return Vec3{X: float64(r.Ax), Y: float64(r.Ay), Z: float64(r.Az)}
		})
		if e != nil {
			return Vec3{}, Vec3{}, 0, nil, e
		}

		c := stillnessConfidence(stats.StdDev)
		data[p] = poseData{pose: p, mean: stats.Mean, std: stats.StdDev, conf: c}
		poseStats = append(poseStats, AccelPoseStats{
			Pose:        p,
			Samples:     stats.Samples,
			DurationSec: stats.DurationSec,
			Mean:        stats.Mean,
			StdDev:      stats.StdDev,
			Confidence:  c,
		})

		fmt.Printf("  Pose %s: mean=(%.1f, %.1f, %.1f) std=(%.1f, %.1f, %.1f) conf=%.2f\n",
			p, stats.Mean.X, stats.Mean.Y, stats.Mean.Z, stats.StdDev.X, stats.StdDev.Y, stats.StdDev.Z, c)
	}

	// Compute bias and scale per axis using + and - poses.
	// For axis X:
	//   plus = sx*(+G) + bx
	//   minus = sx*(-G) + bx
	// => bx = (plus + minus)/2
	// => sx*G = (plus - minus)/2
	// We do not know absolute G in counts; we compute a reference Graw as average of the three axes.
	px := data["+X"].mean.X
	mx := data["-X"].mean.X
	py := data["+Y"].mean.Y
	my := data["-Y"].mean.Y
	pz := data["+Z"].mean.Z
	mz := data["-Z"].mean.Z

	bias = Vec3{
		X: (px + mx) / 2,
		Y: (py + my) / 2,
		Z: (pz + mz) / 2,
	}

	gx := math.Abs((px - mx) / 2)
	gy := math.Abs((py - my) / 2)
	gz := math.Abs((pz - mz) / 2)

	// Robust reference magnitude (average; could use median)
	gRef := (gx + gy + gz) / 3
	if gRef < 1 {
		return Vec3{}, Vec3{}, 0, poseStats, errors.New("accelerometer calibration failed: insufficient gravity separation (gRef too small)")
	}

	// scale in counts per "gRef"; so corrected = (raw - bias)/scale yields ~[-1..1] in "gRef units"
	scale = Vec3{
		X: gx / gRef,
		Y: gy / gRef,
		Z: gz / gRef,
	}
	// Convert to direct divisor for each axis (so corrected ~ (raw-bias)/(gx) * gRef) – store as counts-per-gRef
	// We store "counts per gRef" so later: corrected = (raw-bias)/(scaleCounts); where scaleCounts = gx (etc)
	// To avoid confusion, store scaleCounts directly:
	scale = Vec3{X: gx, Y: gy, Z: gz}

	// Confidence: combine pose stillness confidences and gravity consistency
	poseConf := 0.0
	for _, p := range poses {
		poseConf += data[p].conf
	}
	poseConf /= float64(len(poses))

	consistency := gravityConsistencyConfidence(gx, gy, gz)
	confidence = clamp01(0.65*poseConf + 0.35*consistency)
	if confidence < confFloor {
		confidence = confFloor
	}
	return bias, scale, confidence, poseStats, nil
}

func gravityConsistencyConfidence(gx, gy, gz float64) float64 {
	m := (gx + gy + gz) / 3
	if m <= 0 {
		return confFloor
	}
	// coefficient of variation
	cv := std3(gx, gy, gz) / m
	// map: cv 0 -> 1.0, cv 0.15 -> ~0.7, cv 0.35 -> ~0.3
	return clamp01(1.0 - (cv / 0.5))
}

// ---------- Guided mag calibration ----------

func guidedMag(in *bufio.Reader, readFn func() (imu.IMURaw, error), maxDur time.Duration) (offset Vec3, scale Vec3, confidence float64, stats PhaseStats, err error) {
	magSamples, st, err := captureUntilEnterOrTimeout(in, readFn, maxDur, func(r imu.IMURaw) Vec3 {
		return Vec3{X: float64(r.Mx), Y: float64(r.My), Z: float64(r.Mz)}
	})
	if err != nil {
		return Vec3{}, Vec3{}, 0, PhaseStats{}, err
	}
	stats = st

	// Min/max per axis
	minV := Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	maxV := Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, s := range magSamples {
		minV.X = math.Min(minV.X, s.X)
		minV.Y = math.Min(minV.Y, s.Y)
		minV.Z = math.Min(minV.Z, s.Z)
		maxV.X = math.Max(maxV.X, s.X)
		maxV.Y = math.Max(maxV.Y, s.Y)
		maxV.Z = math.Max(maxV.Z, s.Z)
	}

	offset = Vec3{
		X: (maxV.X + minV.X) / 2,
		Y: (maxV.Y + minV.Y) / 2,
		Z: (maxV.Z + minV.Z) / 2,
	}
	halfRange := Vec3{
		X: (maxV.X - minV.X) / 2,
		Y: (maxV.Y - minV.Y) / 2,
		Z: (maxV.Z - minV.Z) / 2,
	}

	// Guard
	if halfRange.X < 1 || halfRange.Y < 1 || halfRange.Z < 1 {
		stats.Notes = append(stats.Notes, "insufficient_mag_excitation: rotate more in 3D / move away from metal")
		return offset, Vec3{X: 1, Y: 1, Z: 1}, confFloor, stats, nil
	}

	// Scale: normalize axes to common radius (average half-range)
	rRef := (halfRange.X + halfRange.Y + halfRange.Z) / 3
	scale = Vec3{
		X: halfRange.X / rRef,
		Y: halfRange.Y / rRef,
		Z: halfRange.Z / rRef,
	}
	// Store scale in "counts" half-range as the divisor (like accel): corrected = (raw-offset)/halfRange * rRef
	// For simplicity, store halfRange directly:
	scale = halfRange

	// Confidence based on coverage and sphericity after correction
	coverage := magCoverageConfidence(halfRange)
	sphericity := magSphericityConfidence(magSamples, offset, scale)

	confidence = clamp01(0.55*coverage + 0.45*sphericity)
	if confidence < confFloor {
		confidence = confFloor
	}
	return offset, scale, confidence, stats, nil
}

func magCoverageConfidence(halfRange Vec3) float64 {
	// Encourage balanced excitation across axes
	m := (halfRange.X + halfRange.Y + halfRange.Z) / 3
	if m <= 0 {
		return confFloor
	}
	cv := std3(halfRange.X, halfRange.Y, halfRange.Z) / m
	return clamp01(1.0 - (cv / 0.7))
}

func magSphericityConfidence(samples []Vec3, offset Vec3, halfRange Vec3) float64 {
	// Apply simple correction: (raw-offset)/halfRange (dimensionless) then check norm stability.
	// If rotation covers all orientations, norms should be near-constant.
	n := len(samples)
	if n < 50 {
		return confFloor
	}
	norms := make([]float64, 0, n)
	for _, s := range samples {
		x := (s.X - offset.X) / safeDiv(halfRange.X)
		y := (s.Y - offset.Y) / safeDiv(halfRange.Y)
		z := (s.Z - offset.Z) / safeDiv(halfRange.Z)
		norms = append(norms, math.Sqrt(x*x+y*y+z*z))
	}
	mean, sd := meanStd(norms)
	if mean <= 0 {
		return confFloor
	}
	cv := sd / mean
	// map: cv 0.05 -> ~0.9, cv 0.15 -> ~0.7, cv 0.35 -> ~0.3
	return clamp01(1.0 - (cv / 0.5))
}

// ---------- Sampling helpers ----------

type sample struct {
	T time.Time
	V Vec3
}

func captureSamples(readFn func() (imu.IMURaw, error), dur time.Duration, f func(imu.IMURaw) Vec3) ([]Vec3, PhaseStats, error) {
	start := time.Now()
	deadline := start.Add(dur)

	targetPeriod := time.Second / time.Duration(sampleHz)

	var values []Vec3
	for time.Now().Before(deadline) {
		r, err := readFn()
		if err != nil {
			return nil, PhaseStats{}, err
		}
		values = append(values, f(r))
		time.Sleep(targetPeriod)
	}
	stats := computeStats(values, dur)
	return values, stats, nil
}

func captureUntilEnterOrTimeout(in *bufio.Reader, readFn func() (imu.IMURaw, error), maxDur time.Duration, f func(imu.IMURaw) Vec3) ([]Vec3, PhaseStats, error) {
	start := time.Now()
	deadline := start.Add(maxDur)

	// Non-blocking ENTER detector: we start a goroutine waiting for newline
	stopCh := make(chan struct{}, 1)
	go func() {
		_, _ = in.ReadString('\n')
		stopCh <- struct{}{}
	}()

	targetPeriod := time.Second / time.Duration(sampleHz)

	var values []Vec3
	for {
		select {
		case <-stopCh:
			dur := time.Since(start)
			stats := computeStats(values, dur)
			return values, stats, nil
		default:
			if time.Now().After(deadline) {
				dur := time.Since(start)
				stats := computeStats(values, dur)
				stats.Notes = append(stats.Notes, "stopped_by_timeout")
				return values, stats, nil
			}
			r, err := readFn()
			if err != nil {
				return nil, PhaseStats{}, err
			}
			values = append(values, f(r))
			time.Sleep(targetPeriod)
		}
	}
}

func computeStats(values []Vec3, dur time.Duration) PhaseStats {
	n := len(values)
	if n == 0 {
		return PhaseStats{Samples: 0, DurationSec: dur.Seconds()}
	}
	var sx, sy, sz float64
	var sax, say, saz float64
	for _, v := range values {
		sx += v.X
		sy += v.Y
		sz += v.Z
		sax += math.Abs(v.X)
		say += math.Abs(v.Y)
		saz += math.Abs(v.Z)
	}
	mean := Vec3{X: sx / float64(n), Y: sy / float64(n), Z: sz / float64(n)}
	meanAbs := Vec3{X: sax / float64(n), Y: say / float64(n), Z: saz / float64(n)}

	var vx, vy, vz float64
	for _, v := range values {
		dx := v.X - mean.X
		dy := v.Y - mean.Y
		dz := v.Z - mean.Z
		vx += dx * dx
		vy += dy * dy
		vz += dz * dz
	}
	std := Vec3{
		X: math.Sqrt(vx / float64(n)),
		Y: math.Sqrt(vy / float64(n)),
		Z: math.Sqrt(vz / float64(n)),
	}

	return PhaseStats{
		Samples:     n,
		DurationSec: dur.Seconds(),
		Mean:        mean,
		MeanAbs:     meanAbs,
		StdDev:      std,
	}
}

func integrate(values []Vec3) Vec3 {
	// Best-effort integration assuming uniform sampling at sampleHz.
	// (For calibration quality/bias refinement this is acceptable.)
	if len(values) == 0 {
		return Vec3{}
	}
	dt := 1.0 / float64(sampleHz)
	var ix, iy, iz float64
	for _, v := range values {
		ix += v.X * dt
		iy += v.Y * dt
		iz += v.Z * dt
	}
	return Vec3{X: ix, Y: iy, Z: iz}
}

// ---------- Confidence heuristics ----------

func stillnessConfidence(std Vec3) float64 {
	// Use average std dev across axes.
	s := (std.X + std.Y + std.Z) / 3
	switch {
	case s <= stillStdGood:
		return 1.0
	case s >= stillStdBad:
		return confFloor
	default:
		// Linear interpolation between good and bad
		t := (s - stillStdGood) / (stillStdBad - stillStdGood)
		return clamp01(1.0 - 0.95*t)
	}
}

func rotationConfidence(axis string, st PhaseStats) float64 {
	dom := dominantForAxis(axis, st.AxisDominance)
	meanAbs := meanAbsForAxis(axis, st.MeanAbs)

	// Duration factor
	durFactor := clamp01(st.DurationSec / gyroRotMinDur.Seconds())
	if st.DurationSec > gyroRotMaxDur.Seconds() {
		durFactor = 1
	}

	// Dominance factor
	var domFactor float64
	switch {
	case dom >= dominanceGood:
		domFactor = 1
	case dom <= dominanceBad:
		domFactor = 0.2
	default:
		t := (dom - dominanceBad) / (dominanceGood - dominanceBad)
		domFactor = 0.2 + 0.8*clamp01(t)
	}

	// Rotation magnitude factor
	rateFactor := 0.2
	if meanAbs >= minMeanAbsRate {
		// Let it grow to 1.0 by ~4x threshold
		rateFactor = clamp01(meanAbs / (4 * minMeanAbsRate))
	}
	conf := 0.25*durFactor + 0.45*domFactor + 0.30*rateFactor
	return clamp01(max(conf, confFloor))
}

func axisDominance(meanAbs Vec3) Vec3 {
	sum := meanAbs.X + meanAbs.Y + meanAbs.Z
	if sum <= 0 {
		return Vec3{}
	}
	return Vec3{
		X: meanAbs.X / sum,
		Y: meanAbs.Y / sum,
		Z: meanAbs.Z / sum,
	}
}

func dominantForAxis(axis string, dom Vec3) float64 {
	switch axis {
	case "x":
		return dom.X
	case "y":
		return dom.Y
	case "z":
		return dom.Z
	default:
		return 0
	}
}

func meanAbsForAxis(axis string, v Vec3) float64 {
	switch axis {
	case "x":
		return v.X
	case "y":
		return v.Y
	case "z":
		return v.Z
	default:
		return 0
	}
}

func overallConfidence(gyroStatic, gyroRot, accel6, mag float64) float64 {
	// Weighted; gyro static is foundational, mag matters for yaw.
	wGS, wGR, wA, wM := 0.20, 0.20, 0.25, 0.35
	return clamp01(wGS*gyroStatic + wGR*gyroRot + wA*accel6 + wM*mag)
}

// ---------- Output ----------

func writeResult(res CalibrationResult) error {
	ts := time.Now().Format("2006-01-02T15-04-05Z07-00")
	name := fmt.Sprintf("%s_%s_inertial_calibration.json", res.IMU, ts)

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(name, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("\nWrote: %s\n", name)
	return nil
}

// ---------- Console helpers ----------

func waitEnter(in *bufio.Reader, prompt string) {
	fmt.Print(prompt)
	_, _ = in.ReadString('\n')
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	os.Exit(1)
}

// ---------- Small math helpers ----------

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func safeDiv(x float64) float64 {
	if math.Abs(x) < 1e-9 {
		if x >= 0 {
			return 1e-9
		}
		return -1e-9
	}
	return x
}

func meanStd(xs []float64) (mean float64, sd float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	for _, v := range xs {
		mean += v
	}
	mean /= float64(len(xs))
	var s float64
	for _, v := range xs {
		d := v - mean
		s += d * d
	}
	sd = math.Sqrt(s / float64(len(xs)))
	return mean, sd
}

func std3(a, b, c float64) float64 {
	m := (a + b + c) / 3
	return math.Sqrt(((a-m)*(a-m) + (b-m)*(b-m) + (c-m)*(c-m)) / 3)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
