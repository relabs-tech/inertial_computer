package app

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

func RunWeb() error {
	var (
		mu       sync.RWMutex
		lastPose orientation.Pose
		havePose bool
	)

	// 1) Connect to MQTT broker on the Pi
	opts := mqtt.NewClientOptions().
		AddBroker("tcp://localhost:1883").
		SetClientID("inertial-web-subscriber")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Println("connected to MQTT broker at tcp://localhost:1883")

	// 2) Subscribe to pose topic and update lastPose on each message
	token := client.Subscribe("inertial/pose", 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p orientation.Pose
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			log.Printf("MQTT payload unmarshal error: %v", err)
			return
		}
		mu.Lock()
		lastPose = p
		havePose = true
		mu.Unlock()
	})
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	log.Println("subscribed to MQTT topic inertial/pose")

	// 3) JSON API endpoint: latest pose
	http.HandleFunc("/api/orientation", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !havePose {
			http.Error(w, "no data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastPose); err != nil {
			log.Printf("json encode error: %v", err)
		}
	})

	// 4) Static files from ./web as the root
	fs := http.FileServer(http.Dir("web"))
	http.Handle("/", fs)

	addr := ":8080"
	log.Printf("web server listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
