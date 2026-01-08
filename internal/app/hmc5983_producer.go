// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
	"periph.io/x/devices/v3/hmc5983"
)

// hmcPayload is the JSON schema we publish.
// mx,my,mz are in µT×10 (int16) to match project conventions.
// norm is optional magnitude in µT.
// time is RFC3339.
type hmcPayload struct {
	Mx   int16   `json:"mx"`
	My   int16   `json:"my"`
	Mz   int16   `json:"mz"`
	Norm float64 `json:"norm"`
	Time string  `json:"time"`
}

func RunHMC5983Producer() {
	// Load config.
	if err := config.InitGlobal("./inertial_config.txt"); err != nil {
		fmt.Printf("hmc: config init failed: %v\n", err)
		return
	}
	cfg := config.Get()

	// Initialize periph host.
	if _, err := host.Init(); err != nil {
		fmt.Printf("hmc: periph host init failed: %v\n", err)
		return
	}

	// Open I2C bus.
	busName := fmt.Sprintf("%d", cfg.HMCI2CBus)
	if busName == "0" || busName == "" {
		busName = "1"
	}
	bus, err := i2creg.Open(busName)
	if err != nil {
		fmt.Printf("hmc: i2c open failed on bus %s: %v\n", busName, err)
		return
	}
	defer bus.Close()

	// Parse HMC options from config file lines (simple helper reads env-like via config file not exposed here).
	addr := cfg.HMCI2CAddr
	if addr == 0 { addr = 0x1E }
	odr := cfg.HMCODRHz
	if odr == 0 { odr = 15 }
	avg := cfg.HMCAvgSamples
	if avg == 0 { avg = 1 }
	gain := cfg.HMCGainCode
	mode := cfg.HMCMode
	if mode == "" { mode = "continuous" }
	// Create device.
	dev, err := hmc5983.New(bus, hmc5983.Opts{Addr: addr, ODRHz: odr, AvgSamples: avg, GainCode: gain, Mode: mode})
	if err != nil {
		fmt.Printf("hmc: init failed: %v\n", err)
		return
	}
	ida, idb, idc, _ := dev.ID()
	fmt.Printf("[HMC] ID=%q %q %q (addr=0x%X)\n", ida, idb, idc, addr)

	// MQTT client.
	clientID := cfg.MQTTClientIDHMC
	if clientID == "" {
		clientID = "inertial-hmc-producer"
	}
	opts := mqtt.NewClientOptions().AddBroker(cfg.MQTTBroker).SetClientID(clientID)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("hmc: mqtt connect error: %v\n", token.Error())
		return
	}
	defer client.Disconnect(250)

	topic := cfg.TopicMagHMC
	if topic == "" {
		topic = "inertial/mag/hmc"
	}

	ms := cfg.HMCSampleInterval
	if ms <= 0 { ms = 100 }
	interval := time.Duration(ms) * time.Millisecond
	// Start loop.
	fmt.Println("hmc: producer started")
	for {
		x, y, z, err := dev.Sense()
		if err != nil {
			fmt.Printf("hmc: read error: %v\n", err)
			time.Sleep(interval)
			continue
		}
		// Compute magnitude in µT (float).
		mx := float64(x) / 10.0
		my := float64(y) / 10.0
		mz := float64(z) / 10.0
		norm := (mx*mx + my*my + mz*mz)
		norm = sqrt(norm)
		payload := hmcPayload{Mx: x, My: y, Mz: z, Norm: norm, Time: time.Now().UTC().Format(time.RFC3339)}
		b, _ := json.Marshal(payload)
		t := client.Publish(topic, 0, false, b)
		t.Wait()
		// brief sleep
		time.Sleep(interval)
	}
}

func sqrt(x float64) float64 {
	// Simple Newton method for sqrt to avoid extra deps.
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}
