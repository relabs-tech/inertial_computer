package app

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/env"
	"github.com/relabs-tech/inertial_computer/internal/gps"
	imu_raw "github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunWeb() error {
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
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-web-subscriber")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Println("web: connected to MQTT broker at tcp://localhost:1883")

	// 2) Subscribe to inertial/pose
	poseToken := client.Subscribe("inertial/pose", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to MQTT topic inertial/pose")

	// 3) Subscribe to inertial/gps
	gpsToken := client.Subscribe("inertial/gps", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to MQTT topic inertial/gps")

	// 4) Subscribe to inertial/pose/fused
	fusedToken := client.Subscribe("inertial/pose/fused", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to MQTT topic inertial/pose/fused")

	// 4b) inertial/imu/left
	imuLeftToken := client.Subscribe("inertial/imu/left", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to inertial/imu/left")

	// 4c)inertial/imu/right
	imuRightToken := client.Subscribe("inertial/imu/right", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to inertial/imu/right")

	// 4d)inertial/bmp/left
	envLeftToken := client.Subscribe("inertial/bmp/left", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to inertial/bmp/left")

	// 4e) inertial/bmp/right
	envRightToken := client.Subscribe("inertial/bmp/right", 0, func(_ mqtt.Client, msg mqtt.Message) {
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
	log.Println("web: subscribed to inertial/bmp/right")

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

	addr := ":8080"
	log.Printf("web: listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
