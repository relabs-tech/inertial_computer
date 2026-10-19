// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// ChronySource is one row of `chronyc sources` output (one time reference
// chrony is comparing against, e.g. the PPS or NMEA GPS refclocks).
type ChronySource struct {
	Mode       string `json:"mode"`  // '#'=refclock, '^'=server, '='=peer
	State      string `json:"state"` // '*'=selected, '+'=combined, '?'=unreachable, ...
	Name       string `json:"name"`
	Stratum    int    `json:"stratum"`
	Poll       int    `json:"poll"`
	Reach      int    `json:"reach"` // octal reachability register, printed as decimal digits by chronyc
	LastRx     string `json:"last_rx"`
	LastSample string `json:"last_sample"`
}

// ChronyStatus is the full timing-sources snapshot returned by /api/timing.
type ChronyStatus struct {
	Sources  []ChronySource    `json:"sources"`
	Tracking map[string]string `json:"tracking"`
	Error    string            `json:"error,omitempty"`
}

// parseChronySources parses the table printed by `chronyc -n sources`.
func parseChronySources(output string) []ChronySource {
	var sources []ChronySource
	started := false
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "===") {
			started = true
			continue
		}
		if !started || len(line) < 3 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line[2:])
		if len(fields) < 5 {
			continue
		}
		stratum, _ := strconv.Atoi(fields[1])
		poll, _ := strconv.Atoi(fields[2])
		reach, _ := strconv.Atoi(fields[3])
		sources = append(sources, ChronySource{
			Mode:       string(line[0]),
			State:      string(line[1]),
			Name:       fields[0],
			Stratum:    stratum,
			Poll:       poll,
			Reach:      reach,
			LastRx:     fields[4],
			LastSample: strings.Join(fields[5:], " "),
		})
	}
	return sources
}

// parseChronyTracking parses the "Key : value" lines printed by `chronyc tracking`.
func parseChronyTracking(output string) map[string]string {
	tracking := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key != "" {
			tracking[key] = val
		}
	}
	return tracking
}

// getChronyStatus shells out to chronyc to read live timing-source state.
// Degrades gracefully (returns an Error field) when chronyc/chronyd aren't
// available, e.g. when running off the Raspberry Pi.
func getChronyStatus() ChronyStatus {
	var status ChronyStatus

	srcOut, err := exec.Command("chronyc", "-n", "sources").Output()
	if err != nil {
		status.Error = fmt.Sprintf("chronyc sources: %v", err)
		return status
	}
	status.Sources = parseChronySources(string(srcOut))

	trkOut, err := exec.Command("chronyc", "tracking").Output()
	if err != nil {
		status.Error = fmt.Sprintf("chronyc tracking: %v", err)
		return status
	}
	status.Tracking = parseChronyTracking(string(trkOut))

	return status
}

// HandleTiming serves live chrony timing-source status as JSON for the
// timing.html page.
func HandleTiming(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(getChronyStatus()); err != nil {
		log.Printf("web: timing JSON encode error: %v", err)
	}
}
