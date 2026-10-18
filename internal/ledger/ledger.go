// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

// Package ledger is an append-only audit log: calibration changes, config changes,
// service start/stop, power events, file/ephemeris updates. Each entry is one JSON
// line, so the file can be tailed, grepped, or parsed without a loader.
package ledger

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Entry is a single ledger record.
type Entry struct {
	Timestamp time.Time         `json:"timestamp"`
	Service   string            `json:"service"`
	Event     string            `json:"event"`
	Message   string            `json:"message,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

var (
	mu   sync.Mutex
	path = "./inertial_ledger.jsonl"
)

// Init sets the ledger file path. Optional — Log() works with the default path even
// if Init is never called, so a missing call never silently drops events.
func Init(filePath string) {
	if filePath == "" {
		return
	}
	mu.Lock()
	path = filePath
	mu.Unlock()
}

// Log appends one entry. Failures are swallowed (not returned, not panicked) since
// the ledger must never be able to take down the service recording to it.
func Log(service, event, message string, fields map[string]string) {
	entry := Entry{
		Timestamp: time.Now().UTC(),
		Service:   service,
		Event:     event,
		Message:   message,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	mu.Lock()
	filePath := path
	mu.Unlock()

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}
