// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

// Package pisugar is a minimal I2C driver for the PiSugar 3 / 3 Plus / 3 Air UPS,
// covering only what the reliability subsystem needs: battery voltage/percentage,
// external power detection, and the onboard hardware watchdog. Register map from
// https://docs.pisugar.com/docs/product-wiki/battery/pisugar3/pisugar-3-i2c
package pisugar

import (
	"fmt"

	"periph.io/x/conn/v3/i2c"
)

// DefaultAddr is the PiSugar 3's factory I2C address.
const DefaultAddr uint16 = 0x57

const (
	regCtrl1       = 0x02 // bit7 ext-power-connected(ro), bit6 charging-on, bit2 output-on
	regCtrl2       = 0x03 // bit6 auto-hibernate, bit4 soft-shutdown-enable, bit3 soft-shutdown-pending
	regWriteProt   = 0x0B // write 0x29 to unlock writes, anything else to lock
	regWatchdogCtl = 0x06 // bit7 sw-watchdog-enable, bit5 sw-watchdog-kick
	regWatchdogInt = 0x07 // interval = value * 2s
	regVoltageHi   = 0x22
	regVoltageLo   = 0x23
	regPercent     = 0x2A

	unlockValue = 0x29
	lockValue   = 0xFF
)

// Dev wraps an I2C connection to a PiSugar 3.
type Dev struct {
	conn *i2c.Dev
}

// New opens a PiSugar 3 at addr (0 selects DefaultAddr) on the given bus.
func New(bus i2c.Bus, addr uint16) *Dev {
	if addr == 0 {
		addr = DefaultAddr
	}
	return &Dev{conn: &i2c.Dev{Bus: bus, Addr: addr}}
}

func (d *Dev) readReg(reg byte) (byte, error) {
	buf := make([]byte, 1)
	if err := d.conn.Tx([]byte{reg}, buf); err != nil {
		return 0, fmt.Errorf("pisugar: read reg 0x%02X: %w", reg, err)
	}
	return buf[0], nil
}

func (d *Dev) writeReg(reg, value byte) error {
	if err := d.conn.Tx([]byte{reg, value}, nil); err != nil {
		return fmt.Errorf("pisugar: write reg 0x%02X: %w", reg, err)
	}
	return nil
}

// withUnlocked unlocks the write-protected register range, runs fn, then re-locks —
// even if fn fails, so a transient error can't leave the device permanently writable.
func (d *Dev) withUnlocked(fn func() error) error {
	if err := d.writeReg(regWriteProt, unlockValue); err != nil {
		return err
	}
	fnErr := fn()
	lockErr := d.writeReg(regWriteProt, lockValue)
	if fnErr != nil {
		return fnErr
	}
	return lockErr
}

// VoltageMV returns the battery voltage in millivolts.
func (d *Dev) VoltageMV() (int, error) {
	hi, err := d.readReg(regVoltageHi)
	if err != nil {
		return 0, err
	}
	lo, err := d.readReg(regVoltageLo)
	if err != nil {
		return 0, err
	}
	return int(hi)<<8 | int(lo), nil
}

// Percent returns the PiSugar's own calculated battery percentage (0-100).
func (d *Dev) Percent() (float64, error) {
	v, err := d.readReg(regPercent)
	if err != nil {
		return 0, err
	}
	return float64(v), nil
}

// ExternalPowerConnected reports whether the UPS detects external (shore) power.
func (d *Dev) ExternalPowerConnected() (bool, error) {
	v, err := d.readReg(regCtrl1)
	if err != nil {
		return false, err
	}
	return v&0x80 != 0, nil
}

// Charging reports whether the battery is currently charging (regCtrl1 bit6).
func (d *Dev) Charging() (bool, error) {
	v, err := d.readReg(regCtrl1)
	if err != nil {
		return false, err
	}
	return v&0x40 != 0, nil
}

// OutputOn reports whether the UPS is currently supplying 5V output (regCtrl1 bit2).
func (d *Dev) OutputOn() (bool, error) {
	v, err := d.readReg(regCtrl1)
	if err != nil {
		return false, err
	}
	return v&0x04 != 0, nil
}

// AutoHibernate reports whether auto-hibernate is enabled (regCtrl2 bit6): the UPS
// will hibernate the Pi automatically when the battery drops to its hibernate threshold.
func (d *Dev) AutoHibernate() (bool, error) {
	v, err := d.readReg(regCtrl2)
	if err != nil {
		return false, err
	}
	return v&0x40 != 0, nil
}

// SoftShutdownEnabled reports whether soft shutdown is enabled (regCtrl2 bit4).
func (d *Dev) SoftShutdownEnabled() (bool, error) {
	v, err := d.readReg(regCtrl2)
	if err != nil {
		return false, err
	}
	return v&0x10 != 0, nil
}

// SoftShutdownPending reports whether a soft shutdown is in progress (regCtrl2 bit3) —
// the UPS is actively cutting output to power down the Pi safely.
func (d *Dev) SoftShutdownPending() (bool, error) {
	v, err := d.readReg(regCtrl2)
	if err != nil {
		return false, err
	}
	return v&0x08 != 0, nil
}

// WatchdogArmed reports whether the PiSugar's own power-cycling hardware watchdog
// is currently enabled (regWatchdogCtl bit7).
func (d *Dev) WatchdogArmed() (bool, error) {
	v, err := d.readReg(regWatchdogCtl)
	if err != nil {
		return false, err
	}
	return v&0x80 != 0, nil
}

// EnableHardwareWatchdog arms the PiSugar's own power-cycling watchdog: if it isn't
// kicked within intervalSeconds (rounded to the nearest 2s step), the PiSugar cuts
// and restores 5V output, hard-power-cycling the Pi even if it's fully locked up —
// a stronger guarantee than a systemd-only watchdog, which can't recover a hung kernel.
func (d *Dev) EnableHardwareWatchdog(intervalSeconds int) error {
	steps := byte(intervalSeconds / 2)
	if steps == 0 {
		steps = 1
	}
	return d.withUnlocked(func() error {
		if err := d.writeReg(regWatchdogInt, steps); err != nil {
			return err
		}
		ctl, err := d.readReg(regWatchdogCtl)
		if err != nil {
			return err
		}
		return d.writeReg(regWatchdogCtl, ctl|0x80)
	})
}

// DisableHardwareWatchdog turns the PiSugar power-cycling watchdog back off.
func (d *Dev) DisableHardwareWatchdog() error {
	return d.withUnlocked(func() error {
		ctl, err := d.readReg(regWatchdogCtl)
		if err != nil {
			return err
		}
		return d.writeReg(regWatchdogCtl, ctl&^byte(0x80))
	})
}

// KickHardwareWatchdog resets the PiSugar's watchdog timer ("feeds the dog").
func (d *Dev) KickHardwareWatchdog() error {
	return d.withUnlocked(func() error {
		ctl, err := d.readReg(regWatchdogCtl)
		if err != nil {
			return err
		}
		return d.writeReg(regWatchdogCtl, ctl|0x20)
	})
}
