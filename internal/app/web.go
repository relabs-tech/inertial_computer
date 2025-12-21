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
		mu       sync.RWMutex
		lastPose orientation.Pose
		havePose bool

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

	// 2) Subscribe to pose
	poseToken := client.Subscribe(cfg.TopicPose, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("web: pose unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastPose = p
		havePose = true
		mu.Unlock()
	})
	poseToken.Wait()
	if poseToken.Error() != nil {
		return poseToken.Error()
	}
	log.Printf("web: subscribed to MQTT topic %s", cfg.TopicPose)

	// 3) Subscribe to GPS
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

	// 5) JSON API: latest pose
	http.HandleFunc("/api/orientation", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !havePose {
			http.Error(w, "no orientation data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastPose); err != nil {
			log.Printf("web: orientation JSON encode error: %v", err)
		}
	})

	// 5b) JSON API: latest fused pose
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

	// 7) Static UI from ./web
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", cfg.WebServerPort)
	log.Printf("web: listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
