# Marine/Aviation Navigation Computer — Vision & Action List

This document captures the plan discussed for evolving Inertial Computer into a ruggedized
dead-reckoning / celestial / GPS navigation computer and marine chronometer, usable on a boat
or an aircraft. It is a living backlog — update it as decisions are made and phases complete.

---

## 1. Vision

- Primary navigation method: **dead reckoning** from dual IMUs + compass (heading) combined with
  a speed input (GPS speed, NMEA0183 log/paddlewheel sensor, or manual entry — all three should
  be supported as interchangeable speed sources).
- **Position fixes** from GPS (when available) and from celestial body sights (entered via the
  `celestial/` app) correct/reset the DR position.
- **GPS also serves as time correction** — the system acts as a marine chronometer, disciplining
  system/RTC time when GPS is locked and tracking drift rate when it isn't.
- Platform: **both boat and aircraft**, switchable (different speed source semantics: SOG/STW for
  marine, airspeed/ground speed for aviation).
- New apps, implemented as **tabs on the celestial app page**: Shiplog, Passage Plan, Chronometer.
- Must **minimise current consumption** (10 Ah UPS battery budget) while being extremely reliable:
  **must recover from any fault as long as it has power**, and auto-start everything on boot.
- Must **regularly update** any files that go stale (ephemeris/leap-second data, HP48 programs,
  etc.) and **keep a ledger/log** of these updates and other key system events.
- Local access via HDMI screen + keyboard/mouse, **and** via a Wi-Fi hotspot for browsing the web
  apps from a laptop/tablet.
- Optional **HP48GX** connection for manual nav calculations, with HP programs kept up to date.
- **Skyfield** (Python) will be installed, presumably for higher-precision/offline ephemeris
  computation alongside (or validating) the JS celestial engine.

---

## 2. Hardware gap analysis

Current hardware: Raspberry Pi 4, dual MPU9250 IMUs, HMC5983 compass, dual BMP env sensors, GPS
(NMEA), SSD1306 OLED displays, 10 Ah UPS (model unconfirmed — "Pilipane woqbpeng9i", likely a
generic/rebranded clone UPS HAT).

| Gap | Why it's needed | Status |
|---|---|---|
| **RTC with battery backup** (e.g. DS3231) | Pi 4 has no onboard RTC; without one, clock resets to boot time on power loss — breaks chronometer function and DR timestamping before GPS lock. | **Needed** |
| **UPS with I2C fuel gauge + auto-power-on-restore** | Required to know battery state, safely shut down before brownout, and auto-boot when power returns. | Have a UPS (10 Ah) but **exact chipset/I2C address unconfirmed** — see open question below |
| **WiFi for hotspot** | Needed for laptop/tablet access to web apps. | **Already available** — Pi 4 has onboard WiFi/BT |
| **External/active GPS antenna** | Cabin/below-deck reception is often poor; GPS is also the time source. | Recommended, likely already planned |
| **Speed input for DR** | IMU+compass alone cannot do dead reckoning (accelerometers can't sense constant velocity). Need GPS speed, NMEA log/paddlewheel sensor, or manual entry. | Plan to support **all three** |
| **Storage reliability** | SD cards are the #1 failure point under vibration + sudden power loss. | Recommend industrial-grade SD card or eMMC/SSD + read-only/overlay root filesystem |
| **HP48GX serial link** | HP48GX's wired port is not directly RS232/USB compatible. | Need HP48 serial cable + USB-serial adapter (or IR dongle, less reliable) |
| **Compass placement** | Metal hull/airframe biases magnetometer readings. | Plan physical placement away from engines/metal mass; in-situ hard/soft-iron calibration required |

### Open question: UPS fuel-gauge chip
"Pilipane woqbpeng9i" is a generic/reseller product name, not a chip identifier. Need one of:
- `sudo i2cdetect -y 1` output (hex address(es) found — common: `0x36` MAX1704x, `0x40-0x45` INA219/INA226)
- A product link/wiki page, or chip markings on the board
- Fallback if unresolved: implement a generic INA219/INA226 voltage+current reader with
  configurable address, since that's the most common chip in clone UPS HATs.

---

## 3. Software architecture additions

1. **Dead reckoning producer** (`cmd/dr_producer`) — fuses gyro+mag heading with a pluggable speed
   source (GPS speed / NMEA log sensor / manual entry), integrates position, publishes
   `inertial/nav/dr` with explicit uncertainty/fix-age. Must degrade gracefully, never crash, when
   the speed input is missing.
2. **Navigation fusion layer** — merges DR + GPS fixes + manually-entered celestial LOPs into a
   single best-position estimate (start simple: weighted average / basic Kalman filter).
3. **Chronometer service** — disciplines system/RTC time from GPS when locked, tracks and logs
   drift rate when not, exposes "best UTC estimate + confidence" over MQTT/REST.
4. **Shiplog / Passage Plan / Chronometer UI tabs** — new panels alongside `celestial/index.html`,
   backed by small REST endpoints (reuse `cmd/celestial` server or a new `cmd/navlog` service).
5. **Ledger/audit log** (`internal/ledger`) — append-only JSONL log recording calibration changes,
   config changes, ephemeris/file updates, service restarts, power events.
6. **Auto-updater service** — scheduled (systemd timer) job refreshing anything that goes stale
   (leap-second data, Skyfield ephemeris kernels, HP48 program sync on cable-connect), logging
   every attempt/result to the ledger. **Pre-bundle the Skyfield ephemeris kernel** rather than
   relying on first-run internet download (often offline underway).
7. **Reliability hardening**:
   - `internal/watchdog` — minimal systemd `sd_notify`/watchdog client (no new dependency).
   - Hardened `systemd/` unit files for every producer/consumer (`Type=notify`, `WatchdogSec`,
     `Restart=always`, proper ordering), grouped under one `inertial.target`.
   - `scripts/setup_rtc.sh` — DS3231 overlay + `fake-hwclock` disable + hwclock sync.
   - `scripts/setup_readonly_root.sh` — enable Raspberry Pi OS overlay root filesystem.
   - Battery/UPS monitor (`cmd/power_producer`) — polls fuel gauge, publishes to MQTT, logs
     low/critical battery transitions to the ledger, triggers configurable safe-shutdown command.
8. **HP48GX sync tool** — small separate utility (likely Python/pyserial; Kermit/XMODEM tooling
   for HP48 is mature in that ecosystem), triggered manually or on cable-connect, logged to ledger.
9. **Skyfield integration** — Python environment/venv on the Pi, pre-bundled ephemeris kernel,
   likely its own small service producing/cross-checking almanac data for the celestial app.

---

## 4. Action list (phased)

### Phase 1 — Reliability hardening (in progress / next up)
- [ ] `internal/ledger` package (append-only JSONL, `Log(service, event, message, fields)`)
- [ ] `internal/watchdog` package (sd_notify `Ready()` + `StartHeartbeat()`, no new Go dependency)
- [ ] Config additions: `LEDGER_FILE_PATH`, `POWER_I2C_BUS`, `POWER_I2C_ADDR`,
      `POWER_POLL_INTERVAL_MS`, `POWER_LOW_BATTERY_PERCENT`, `POWER_CRITICAL_BATTERY_PERCENT`,
      `POWER_SHUTDOWN_COMMAND`, `TOPIC_POWER`, `MQTT_CLIENT_ID_POWER`
- [ ] Resolve UPS fuel-gauge chip (see open question above), then implement
      `internal/sensors/power.go` + `internal/app/power_producer.go` + `cmd/power_producer`
- [ ] Wire ledger + watchdog heartbeat into: `imu_producer`, `gps_producer`, `hmc5983_producer`,
      `web`, `display`, `celestial`, `console_mqtt`, `power_producer`
- [ ] `systemd/` directory: hardened unit files + `inertial.target` + `scripts/install_systemd_services.sh`
- [ ] `scripts/setup_rtc.sh` (requires DS3231 hardware)
- [ ] `scripts/setup_readonly_root.sh`
- [ ] Update README/ARCHITECTURE/QUICKSTART/TODO with the reliability subsystem

### Phase 2 — Dead reckoning + navigation fusion
- [ ] Decide/implement speed-source abstraction (GPS speed, NMEA log sensor, manual entry)
- [ ] `cmd/dr_producer` + DR integration math, publishing `inertial/nav/dr`
- [ ] Navigation fusion layer combining DR + GPS + celestial LOPs

### Phase 3 — Chronometer
- [ ] GPS-disciplined time service, drift tracking/logging when GPS unavailable
- [ ] RTC integration (read on boot before GPS lock, write-back when GPS/NTP trusted)

### Phase 4 — New UI: Shiplog / Passage Plan / Chronometer tabs
- [ ] Tab shell around `celestial/index.html` (or sibling pages linked from it)
- [ ] Shiplog: periodic automatic log entries (position, heading, speed, notes)
- [ ] Passage Plan: waypoints/route, track plot
- [ ] Chronometer tab: live best-UTC estimate, drift history

### Phase 5 — Auto-updater + Skyfield + HP48GX
- [ ] Scheduled updater service + ledger integration
- [ ] Skyfield Python environment + pre-bundled ephemeris kernel
- [ ] HP48GX serial cable/adapter wiring + sync utility

---

## 5. Open questions to resolve before/while implementing

- Exact UPS fuel-gauge chip + I2C address (see §2)
- Confirm Pi is Pi 4 (has onboard WiFi — confirmed) and whether a DS3231 (or similar) RTC will be
  added
- Preferred speed-source priority/fallback order for DR (GPS → log sensor → manual?)
- HP48GX cable: wired serial vs IR dongle
