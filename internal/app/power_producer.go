// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/ledger"
	"github.com/relabs-tech/inertial_computer/internal/sensors/pisugar"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

// powerPayload is the JSON schema published to TOPIC_POWER.
type powerPayload struct {
	VoltageMV     int     `json:"voltage_mv"`
	Percent       float64 `json:"percent"`
	ExternalPower bool    `json:"external_power"`
	Time          string  `json:"time"`
}

// battery state transitions, tracked so the ledger only gets one entry per transition
// instead of one per poll.
type batteryState int

const (
	stateNormal batteryState = iota
	stateLow
	stateCritical
)

// RunPowerProducer polls the UPS (PiSugar 3 family) over I2C, publishes battery state
// to MQTT, logs low/critical transitions to the ledger, and optionally arms the UPS's
// own hardware watchdog and triggers a safe shutdown at the critical threshold.
func RunPowerProducer() error {
	cfg := config.Get()
	ledger.Init(cfg.LedgerFilePath)

	if _, err := host.Init(); err != nil {
		return fmt.Errorf("power: periph host init: %w", err)
	}

	busName := fmt.Sprintf("%d", cfg.PowerI2CBus)
	if cfg.PowerI2CBus == 0 {
		busName = "1"
	}
	bus, err := i2creg.Open(busName)
	if err != nil {
		return fmt.Errorf("power: i2c open bus %s: %w", busName, err)
	}
	defer bus.Close()

	dev := pisugar.New(bus, cfg.PowerI2CAddr)

	if cfg.PowerHWWatchdogEnable {
		interval := cfg.PowerHWWatchdogIntervalSec
		if interval <= 0 {
			interval = 60
		}
		if err := dev.EnableHardwareWatchdog(interval); err != nil {
			fmt.Printf("power: warning: failed to enable UPS hardware watchdog: %v\n", err)
		} else {
			ledger.Log("power", "hw_watchdog_enabled", "", map[string]string{"interval_sec": fmt.Sprintf("%d", interval)})
		}
	}

	clientID := cfg.MQTTClientIDPower
	if clientID == "" {
		clientID = "inertial-power-producer"
	}
	opts := mqtt.NewClientOptions().AddBroker(cfg.MQTTBroker).SetClientID(clientID)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("power: mqtt connect: %w", token.Error())
	}
	defer client.Disconnect(250)

	topic := cfg.TopicPower
	if topic == "" {
		topic = "inertial/power"
	}

	pollMS := cfg.PowerPollIntervalMS
	if pollMS <= 0 {
		pollMS = 5000
	}
	interval := time.Duration(pollMS) * time.Millisecond

	lowPct := cfg.PowerLowBatteryPercent
	if lowPct <= 0 {
		lowPct = 20
	}
	criticalPct := cfg.PowerCriticalBatteryPercent
	if criticalPct <= 0 {
		criticalPct = 8
	}

	ledger.Log("power", "start", "power producer started", nil)
	fmt.Println("power: producer started")

	state := stateNormal
	for {
		if cfg.PowerHWWatchdogEnable {
			if err := dev.KickHardwareWatchdog(); err != nil {
				fmt.Printf("power: warning: failed to kick UPS hardware watchdog: %v\n", err)
			}
		}

		mv, err := dev.VoltageMV()
		if err != nil {
			fmt.Printf("power: read voltage error: %v\n", err)
			time.Sleep(interval)
			continue
		}
		pct, err := dev.Percent()
		if err != nil {
			fmt.Printf("power: read percent error: %v\n", err)
			time.Sleep(interval)
			continue
		}
		extPower, err := dev.ExternalPowerConnected()
		if err != nil {
			fmt.Printf("power: read external-power error: %v\n", err)
		}

		payload := powerPayload{
			VoltageMV:     mv,
			Percent:       pct,
			ExternalPower: extPower,
			Time:          time.Now().UTC().Format(time.RFC3339),
		}
		b, _ := json.Marshal(payload)
		client.Publish(topic, 0, false, b).Wait()

		newState := stateNormal
		switch {
		case pct <= criticalPct:
			newState = stateCritical
		case pct <= lowPct:
			newState = stateLow
		}
		if newState != state {
			fields := map[string]string{"percent": fmt.Sprintf("%.1f", pct), "voltage_mv": fmt.Sprintf("%d", mv)}
			switch newState {
			case stateLow:
				ledger.Log("power", "battery_low", "", fields)
			case stateCritical:
				ledger.Log("power", "battery_critical", "", fields)
				triggerShutdown(cfg.PowerShutdownCommand)
			case stateNormal:
				ledger.Log("power", "battery_normal", "", fields)
			}
			state = newState
		}

		time.Sleep(interval)
	}
}

// triggerShutdown runs the configured shutdown command. A blank command (the default)
// disables this — it's an explicit opt-in once the UPS/threshold setup is verified.
func triggerShutdown(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		fmt.Println("power: critical battery, but POWER_SHUTDOWN_COMMAND is empty — not shutting down")
		return
	}
	fmt.Printf("power: critical battery — running shutdown command: %s\n", command)
	ledger.Log("power", "shutdown_triggered", command, nil)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	if err := exec.Command(parts[0], parts[1:]...).Run(); err != nil {
		fmt.Printf("power: shutdown command failed: %v\n", err)
	}
}
