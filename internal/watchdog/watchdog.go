// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

// Package watchdog implements the systemd sd_notify protocol directly (no external
// dependency): READY=1 on startup and periodic WATCHDOG=1 pings so systemd can restart
// a service that hangs, pairing with Type=notify + WatchdogSec in the unit file.
// All functions are no-ops when NOTIFY_SOCKET isn't set (e.g. running manually, not
// under systemd), so every binary stays usable outside of systemd without changes.
package watchdog

import (
	"context"
	"net"
	"os"
	"time"
)

// notify sends a raw sd_notify datagram to $NOTIFY_SOCKET. Returns nil (no-op) if the
// variable isn't set, so callers don't need to special-case non-systemd environments.
func notify(state string) error {
	socketPath := os.Getenv("NOTIFY_SOCKET")
	if socketPath == "" {
		return nil
	}

	addr := &net.UnixAddr{Name: socketPath, Net: "unixgram"}
	conn, err := net.DialUnix("unixgram", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(state))
	return err
}

// Ready tells systemd the service has finished starting (for Type=notify units).
func Ready() {
	_ = notify("READY=1")
}

// StartHeartbeat sends READY=1 once, then sends WATCHDOG=1 on the given interval until
// ctx is canceled. Choose interval as roughly half the unit's WatchdogSec so a single
// missed tick doesn't trigger an unnecessary restart.
func StartHeartbeat(ctx context.Context, interval time.Duration) {
	Ready()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = notify("WATCHDOG=1")
			}
		}
	}()
}

// Stopping tells systemd the service is beginning a graceful shutdown.
func Stopping() {
	_ = notify("STOPPING=1")
}
