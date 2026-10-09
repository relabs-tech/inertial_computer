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
	"github.com/relabs-tech/inertial_computer/internal/gps"
)

// RunCelestial serves the forked "celestial" sight-reduction web app on its
// own port and exposes the latest GPS fix so the app can pre-fill its
// Assumed Position fields, consistent with the message-bus-isolation
// pattern (this consumer never talks to producers directly).
func RunCelestial() error {
	cfg := config.Get()

	var (
		mu      sync.RWMutex
		lastFix gps.Fix
		haveFix bool
	)

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDCelestial)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	gpsToken := client.Subscribe(cfg.TopicGPS, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var f gps.Fix
		if err := json.Unmarshal(msg.Payload(), &f); err != nil {
			log.Printf("celestial: gps unmarshal error: %v", err)
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
	log.Printf("celestial: subscribed to MQTT topic %s", cfg.TopicGPS)

	http.HandleFunc("/api/gps", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		if !haveFix {
			http.Error(w, "no gps data yet", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(lastFix); err != nil {
			log.Printf("celestial: gps JSON encode error: %v", err)
		}
	})

	fs := http.FileServer(http.Dir("celestial"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", cfg.CelestialServerPort)
	log.Printf("celestial: listening on %s", addr)
	return http.ListenAndServe(addr, nil)
}
