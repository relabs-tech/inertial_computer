// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/relabs-tech/inertial_computer/internal/config"
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

// RTCInfo is the hardware RTC (DS3231 on /dev/rtc) reading, read via hwclock.
// chrony's rtcsync writes the PPS-disciplined time back to the RTC, so Drift
// (RTC minus system) should stay small while the GPS PPS lock is good; a growing
// drift means the RTC is running free.
type RTCInfo struct {
	Time    string `json:"time"`     // RTC wall time (UTC)
	SysTime string `json:"sys_time"` // system clock at read time (UTC)
	Drift   string `json:"drift"`    // RTC - system, signed, e.g. "+3 ms"
	Error   string `json:"error,omitempty"`
}

// ChronyStatus is the full timing-sources snapshot returned by /api/timing.
type ChronyStatus struct {
	Sources  []ChronySource    `json:"sources"`
	Tracking map[string]string `json:"tracking"`
	RTC      RTCInfo           `json:"rtc"`
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

// getRTCInfo reads the hardware RTC (DS3231 on /dev/rtc0) via the world-readable
// sysfs interface and compares it to the system clock. The sysfs date/time
// attributes (and /proc/driver/rtc) are readable without root, so the
// unprivileged web service can surface the RTC without hwclock/CAP_SYS_TIME.
// chrony's rtcsync writes the PPS-disciplined time back to the RTC, so Drift
// (RTC minus system) should stay small while the GPS PPS lock is good; a growing
// drift means the RTC is running free. Degrades gracefully when the sysfs node
// is missing (e.g. running off the Raspberry Pi, or a Pi without an RTC).
func getRTCInfo() RTCInfo {
	var info RTCInfo

	dir := config.Get().RTCSysfsDir
	if dir == "" {
		dir = "/sys/class/rtc/rtc0"
	}

	// /sys/class/rtc/rtc0/date -> "YYYY-MM-DD\n", /time -> "HH:MM:SS\n".
	datePart, err := readTrimmed(filepath.Join(dir, "date"))
	if err != nil {
		info.Error = fmt.Sprintf("read rtc date: %v", err)
		return info
	}
	timePart, err := readTrimmed(filepath.Join(dir, "time"))
	if err != nil {
		info.Error = fmt.Sprintf("read rtc time: %v", err)
		return info
	}

	rtc, perr := time.Parse("2006-01-02 15:04:05", datePart+" "+timePart)
	if perr != nil {
		info.Error = fmt.Sprintf("parse rtc time %q %q: %v", datePart, timePart, perr)
		return info
	}

	now := time.Now()
	drift := now.Sub(rtc) // positive => system is ahead of the RTC
	info.SysTime = now.UTC().Format("2006-01-02 15:04:05")
	info.Time = rtc.Format("2006-01-02 15:04:05")
	info.Drift = formatDrift(drift)
	return info
}

// readTrimmed reads a single-value sysfs attribute and trims surrounding
// whitespace/newline.
func readTrimmed(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// formatDrift renders a signed time.Duration as a compact human string, e.g.
// "+12 ms", "-3 s", "450 us".
func formatDrift(d time.Duration) string {
	if d == 0 {
		return "0 s"
	}
	sign := ""
	if d < 0 {
		sign = "-"
		d = -d
	}
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%s%3d us", sign, d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%s%3d ms", sign, d.Milliseconds())
	default:
		sec := d / time.Second
		return fmt.Sprintf("%s%ds", sign, sec)
	}
}

// getChronyStatus shells out to chronyc to read live timing-source state.
// Degrades gracefully (returns an Error field) when chronyc/chronyd aren't
// available, e.g. when running off the Raspberry Pi. The RTC reading is
// gathered independently so it still surfaces when chrony is unavailable.
func getChronyStatus() ChronyStatus {
	var status ChronyStatus

	// Always attempt the RTC read first so it's available regardless of chrony.
	status.RTC = getRTCInfo()

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
