// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/env"
	"github.com/relabs-tech/inertial_computer/internal/gps"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunWeb() error {
	cfg := config.Get()

	var (
		mu           sync.RWMutex
		lastPoseLeft orientation.Pose
		havePoseLeft bool

		lastPoseRight orientation.Pose
		havePoseRight bool

		lastFusedPose orientation.Pose
		haveFusedPose bool

		lastFix gps.Fix
		haveFix bool

		lastIMULeft  imu_raw.IMURaw
		haveIMULeft  bool
		lastIMURight imu_raw.IMURaw
		haveIMURight bool

		lastEnvLeft  env.Sample
		haveEnvLeft  bool
		lastEnvRight env.Sample
		haveEnvRight bool

		lastGPSSatellites struct {
			Satellites []gps.Satellite `json:"satellites"`
			Count      int             `json:"count"`
		}
		haveGPSSatellites bool

		lastGLONASSSatellites struct {
			Satellites []gps.Satellite `json:"satellites"`
			Count      int             `json:"count"`
		}
		haveGLONASSSatellites bool
	)

	// 1) Connect to MQTT
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDWeb)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("web: connected to MQTT broker at %s", cfg.MQTTBroker)

	// 2) Subscribe to left pose
	poseLeftToken := client.Subscribe(cfg.TopicPoseLeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("web: pose left unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastPoseLeft = p
		havePoseLeft = true
		mu.Unlock()
	})
	poseLeftToken.Wait()
	if poseLeftToken.Error() != nil {
		return poseLeftToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicPoseLeft)

	// 3) Subscribe to right pose
	poseRightToken := client.Subscribe(cfg.TopicPoseRight, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("web: pose right unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastPoseRight = p
		havePoseRight = true
		mu.Unlock()
	})
	poseRightToken.Wait()
	if poseRightToken.Error() != nil {
		return poseRightToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicPoseRight)

	// 4) Subscribe to fused pose
	fusedToken := client.Subscribe(cfg.TopicPoseFused, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("web: fused pose unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastFusedPose = p
		haveFusedPose = true
		mu.Unlock()
	})
	fusedToken.Wait()
	if fusedToken.Error() != nil {
		return fusedToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicPoseFused)

	// 5) Subscribe to GPS
	// 5) Subscribe to GPS
	gpsToken := client.Subscribe(cfg.TopicGPS, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var f gps.Fix
		if err := json.Unmarshal(msg.Payload(), &f); err != nil {
			log.Printf("web: gps unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastFix = f
		haveFix = true
		mu.Unlock()
	})
	gpsToken.Wait()
	if gpsToken.Error() != nil {
		return gpsToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicGPS)

	// Subscribe to GPS satellites
	gpsSatToken := client.Subscribe(cfg.TopicGPSSatellites, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var satsData struct {
			Satellites []gps.Satellite `json:"satellites"`
			Count      int             `json:"count"`
		}
		if err := json.Unmarshal(msg.Payload(), &satsData); err != nil {
			log.Printf("web: gps satellites unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastGPSSatellites = satsData
		haveGPSSatellites = true
		mu.Unlock()
	})
	gpsSatToken.Wait()
	if gpsSatToken.Error() != nil {
		return gpsSatToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicGPSSatellites)

	// Subscribe to GLONASS satellites
	glonassSatToken := client.Subscribe(cfg.TopicGLONASSSatellites, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var satsData struct {
			Satellites []gps.Satellite `json:"satellites"`
			Count      int             `json:"count"`
		}
		if err := json.Unmarshal(msg.Payload(), &satsData); err != nil {
			log.Printf("web: glonass satellites unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastGLONASSSatellites = satsData
		haveGLONASSSatellites = true
		mu.Unlock()
	})
	glonassSatToken.Wait()
	if glonassSatToken.Error() != nil {
		return glonassSatToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicGLONASSSatellites)

	// Subscribe to IMU left
	imuLeftToken := client.Subscribe(cfg.TopicIMULeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s imu_raw.IMURaw
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("web: imu left unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastIMULeft = s
		haveIMULeft = true
		mu.Unlock()
	})
	imuLeftToken.Wait()
	if imuLeftToken.Error() != nil {
		return imuLeftToken.Error()
	}
	log.Printf("web: subscribed to %s", cfg.TopicIMULeft)

	// Subscribe to IMU right
	imuRightToken := client.Subscribe(cfg.TopicIMURight, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s imu_raw.IMURaw
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("web: imu right unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastIMURight = s
		haveIMURight = true
		mu.Unlock()
	})
	imuRightToken.Wait()
	if imuRightToken.Error() != nil {
		return imuRightToken.Error()
	}
	log.Printf("web: subscribed to %s", cfg.TopicIMURight)

	// Subscribe to BMP left
	envLeftToken := client.Subscribe(cfg.TopicBMPLeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s env.Sample
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("web: env left unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastEnvLeft = s
		haveEnvLeft = true
		mu.Unlock()
	})
	envLeftToken.Wait()
	if envLeftToken.Error() != nil {
		return envLeftToken.Error()
	}
	log.Printf("web: subscribed to %s", cfg.TopicBMPLeft)

	// 4e) Subscribe to BMP right
	envRightToken := client.Subscribe(cfg.TopicBMPRight, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var s env.Sample
		if err := json.Unmarshal(msg.Payload(), &s); err != nil {
			log.Printf("web: env right unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastEnvRight = s
		haveEnvRight = true
		mu.Unlock()
	})
	envRightToken.Wait()
	if envRightToken.Error() != nil {
		return envRightToken.Error()
	}
	log.Printf("web: subscribed to %s", cfg.TopicBMPRight)

	// 5) JSON API: latest left pose
	http.HandleFunc("/api/orientation/left", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !havePoseLeft {
			http.Error(w, "no left orientation data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastPoseLeft); err != nil {
			log.Printf("web: left orientation JSON encode error: %v", err)
		}
	})

	// 5b) JSON API: latest right pose
	http.HandleFunc("/api/orientation/right", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !havePoseRight {
			http.Error(w, "no right orientation data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastPoseRight); err != nil {
			log.Printf("web: right orientation JSON encode error: %v", err)
		}
	})

	// 5c) JSON API: latest fused pose
	http.HandleFunc("/api/orientation/fused", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !haveFusedPose {
			http.Error(w, "no fused orientation data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastFusedPose); err != nil {
			log.Printf("web: fused orientation JSON encode error: %v", err)
		}
	})

	// 6) JSON API: latest GPS fix
	http.HandleFunc("/api/gps", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !haveFix {
			http.Error(w, "no gps data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastFix); err != nil {
			log.Printf("web: gps JSON encode error: %v", err)
		}
	})

	// 6a) JSON API: GPS satellites
	http.HandleFunc("/api/gps/satellites", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !haveGPSSatellites {
			http.Error(w, "no gps satellites data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastGPSSatellites); err != nil {
			log.Printf("web: gps satellites JSON encode error: %v", err)
		}
	})

	// 6a-2) JSON API: GLONASS satellites
	http.HandleFunc("/api/glonass/satellites", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !haveGLONASSSatellites {
			http.Error(w, "no glonass satellites data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastGLONASSSatellites); err != nil {
			log.Printf("web: glonass satellites JSON encode error: %v", err)
		}
	})

	// 6b) JSON API: latest IMU left/right

	http.HandleFunc("/api/imu/left", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		if !haveIMULeft {
			http.Error(w, "no left imu data yet", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastIMULeft); err != nil {
			log.Printf("web: left imu JSON encode error: %v", err)
		}
	})

	http.HandleFunc("/api/imu/right", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		if !haveIMURight {
			http.Error(w, "no right imu data yet", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastIMURight); err != nil {
			log.Printf("web: right imu JSON encode error: %v", err)
		}
	})

	http.HandleFunc("/api/env/left", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		if !haveEnvLeft {
			http.Error(w, "no left env data yet", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastEnvLeft); err != nil {
			log.Printf("web: left env JSON encode error: %v", err)
		}
	})

	http.HandleFunc("/api/env/right", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		if !haveEnvRight {
			http.Error(w, "no right env data yet", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastEnvRight); err != nil {
			log.Printf("web: right env JSON encode error: %v", err)
		}
	})

	// API endpoint for configuration
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		configData := map[string]interface{}{
			"weather_update_interval_minutes": cfg.WeatherUpdateIntervalMinutes,
		}
		if err := json.NewEncoder(w).Encode(configData); err != nil {
			log.Printf("web: config JSON encode error: %v", err)
		}
	})

	// Calibration WebSocket endpoint
	http.HandleFunc("/api/calibration/ws", HandleCalibrationWS)

	// 7) Static UI from ./web
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", cfg.WebServerPort)
	log.Printf("web: listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
