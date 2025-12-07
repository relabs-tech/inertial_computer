package app

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/relabs-tech/inertial_computer/internal/gps"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunWeb() error {
	var (
		mu       sync.RWMutex
		lastPose orientation.Pose
		havePose bool

		lastFix gps.Fix
		haveFix bool
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

	// 4) JSON API: latest pose
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

	// 5) JSON API: latest GPS fix
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

	// 6) Static UI from ./web
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	addr := ":8080"
	log.Printf("web: listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
