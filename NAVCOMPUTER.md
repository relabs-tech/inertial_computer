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
(NMEA), SSD1306 OLED displays, DS3231 RTC, PiSugar 3 UPS (10 Ah).

| Gap | Why it's needed | Status |
|---|---|---|
| **RTC with battery backup** (e.g. DS3231) | Pi 4 has no onboard RTC; without one, clock resets to boot time on power loss — breaks chronometer function and DR timestamping before GPS lock. | **Done** — DS3231 added, `dtoverlay=i2c-rtc,ds3231` enabled in `config.txt` |
| **GPS PPS (pulse-per-second) discipline** | Raw NMEA timestamps are only ~100ms-accurate; PPS gives sub-microsecond edges for true chronometer-grade GPS time discipline. | **Partially done** — `dtoverlay=pps-gpio,gpiopin=6` enabled, `pps_gpio` module loaded, `/dev/pps0` exists; `gpsd` configured and reading the GPS on `/dev/serial0`; `chrony.conf` has both `refclock SHM`/`PPS` lines. **PPS signal itself is not physically wired yet** — confirmed via `chronyc sources -v` showing 0 reachability on `#? PPS` (expected with no pulse connected). Also added `dtoverlay=uart3` (GPIO4/5, confirmed via `pinctrl`) for a second GPS module. |
| **UPS with I2C fuel gauge + auto-power-on-restore** | Required to know battery state, safely shut down before brownout, and auto-boot when power returns. | **Done** — PiSugar 3 (10 Ah), I2C address `0x57`, register map documented at https://docs.pisugar.com/docs/product-wiki/battery/pisugar3/pisugar-3-i2c. Also has its own hardware watchdog (power-cycles the Pi even on a fully hung kernel) — see `cmd/power_producer`. |
| **WiFi for hotspot** | Needed for laptop/tablet access to web apps. | **Done** — Pi 4 has onboard WiFi/BT, plus a USB WiFi dongle added. **Open item**: the dongle enumerates as `Aicsemi Aic MSC` (mass-storage/"CD-ROM installer" mode) instead of a network interface — typical of cheap AIC8800-chipset dongles needing a mode-switch/driver. Only onboard `wlan0` is active for now. |
| **External/active GPS antenna** | Cabin/below-deck reception is often poor; GPS is also the time source. | Recommended, likely already planned |
| **Speed input for DR** | IMU+compass alone cannot do dead reckoning (accelerometers can't sense constant velocity). Need GPS speed, NMEA log/paddlewheel sensor, or manual entry. | Plan to support **all three** |
| **Storage reliability** | SD cards are the #1 failure point under vibration + sudden power loss. | Recommend industrial-grade SD card or eMMC/SSD + read-only/overlay root filesystem |
| **HP48GX serial link** | HP48GX's wired port is not directly RS232/USB compatible. | Need HP48 serial cable + USB-serial adapter (or IR dongle, less reliable) |
| **Compass placement** | Metal hull/airframe biases magnetometer readings. | Plan physical placement away from engines/metal mass; in-situ hard/soft-iron calibration required |

### Open item: AIC8800 WiFi dongle mode-switch
The USB WiFi dongle shows up as mass storage, not a network adapter. Needs the AIC8800
Linux driver + mode-switch (usb_modeswitch or a vendor tool) before it can run as part of
the hotspot. Not yet investigated in depth — pick up later if the onboard `wlan0` AP isn't
sufficient on its own.

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
   - Battery/UPS monitor (`cmd/power_producer`) — polls PiSugar 3 over I2C (`internal/sensors/pisugar`),
     publishes to MQTT, logs low/critical battery transitions to the ledger, triggers a
     configurable safe-shutdown command, and can optionally arm the PiSugar's own hardware
     watchdog (power-cycles the Pi even on a fully hung kernel).
8. **HP48GX sync tool** — small separate utility (likely Python/pyserial; Kermit/XMODEM tooling
   for HP48 is mature in that ecosystem), triggered manually or on cable-connect, logged to ledger.
9. **Skyfield integration** — Python environment/venv on the Pi, pre-bundled ephemeris kernel,
   likely its own small service producing/cross-checking almanac data for the celestial app.

---

## 4. Action list (phased)

### Phase 1 — Reliability hardening (in progress / next up)
- [x] `internal/ledger` package (append-only JSONL, `Log(service, event, message, fields)`)
- [x] `internal/watchdog` package (sd_notify `Ready()` + `StartHeartbeat()`, no new Go dependency)
- [x] Wire watchdog heartbeat + ledger start-event into: `imu_producer`, `gps_producer`,
      `hmc5983_producer`, `web`, `display`, `celestial`, `console_mqtt`, `power_producer`
- [x] Config additions: `LEDGER_FILE_PATH`, `POWER_I2C_BUS`, `POWER_I2C_ADDR`,
      `POWER_POLL_INTERVAL_MS`, `POWER_LOW_BATTERY_PERCENT`, `POWER_CRITICAL_BATTERY_PERCENT`,
      `POWER_SHUTDOWN_COMMAND`, `TOPIC_POWER`, `MQTT_CLIENT_ID_POWER`,
      `POWER_HW_WATCHDOG_ENABLE`, `POWER_HW_WATCHDOG_INTERVAL_SEC`
- [x] UPS identified as PiSugar 3 (I2C `0x57`); implemented `internal/sensors/pisugar`
      (voltage/percent/external-power + hardware watchdog arm/kick) + `internal/app/power_producer.go`
      + `cmd/power_producer` + `systemd/inertial-power.service`
- [x] `systemd/` directory: hardened unit files (`Type=notify`, `WatchdogSec=30`, `Restart=always`)
      + `inertial.target` + `scripts/install_systemd_services.sh` — units cover imu, gps,
      hmc5983, web, display, celestial, power, and an optional (non-target) console-mqtt debugging unit
- [x] `scripts/setup_rtc.sh` — **turned out to be unnecessary**: verified directly on the Pi that
      `fake-hwclock` was never installed, the DS3231 (`rtc-ds1307` driver) is registered as `/dev/rtc0`
      and used to set the system clock at boot (confirmed in `dmesg`), and `/etc/chrony/chrony.conf`
      already has `rtcsync` so chrony writes the disciplined time back to the RTC itself. No script
      needed — Raspberry Pi OS + chrony's `rtcsync` already do the right thing once a real RTC exists.
- [ ] `scripts/setup_readonly_root.sh`
- [ ] Update README/ARCHITECTURE/QUICKSTART/TODO with the reliability subsystem (in progress)
- [ ] Verify `POWER_LOW_BATTERY_PERCENT`/`POWER_CRITICAL_BATTERY_PERCENT` thresholds against real
      battery behavior before setting `POWER_SHUTDOWN_COMMAND` or `POWER_HW_WATCHDOG_ENABLE=true`
      (both default to safe/off)


### Phase 2 — Dead reckoning + navigation fusion
- [ ] Decide/implement speed-source abstraction (GPS speed, NMEA log sensor, manual entry)
- [ ] `cmd/dr_producer` + DR integration math, publishing `inertial/nav/dr`
- [ ] Navigation fusion layer combining DR + GPS + celestial LOPs

### Phase 3 — Chronometer
- [x] RTC integration verified working: DS3231 sets system clock at boot, chrony's `rtcsync`
      writes back once the clock is disciplined — no custom code/script needed (see Phase 1)
- [x] `gpsd` configured (`DEVICES="/dev/serial0"`, `GPSD_OPTIONS="-n"`, running as `gpsd.service`,
      confirmed reading live NMEA fixes)
- [x] chrony GPS refclocks added to `chrony.conf`:
      `refclock SHM 0 offset 0.0 delay 0.2 refid NMEA noselect` (coarse, always-on fallback once
      offline — confirmed reachable, ~hundreds of ms accuracy, intentionally `noselect`) and
      `refclock PPS /dev/pps0 lock NMEA refid PPS` (precise, sub-microsecond once selected)
- [ ] **PPS not physically wired yet** — `pps_gpio` module is loaded and `/dev/pps0` exists
      (`dtoverlay=pps-gpio,gpiopin=6` in `config.txt`), but with no physical pulse connected,
      chrony's PPS refclock correctly shows 0 reachability. Wire the GPS module's PPS/TIMEPULSE
      output to GPIO6, then re-check `chronyc sources -v` for `#* PPS` to appear.
- [ ] Once PPS is live: double check `/dev/serial0`'s actual UART mapping (`ttyS0` mini-UART vs
      `ttyAMA0` full PL011) — if Bluetooth is enabled, `serial0` may alias to the mini-UART, which
      has clock-scaling-related jitter that can degrade PPS timing precision. Not yet confirmed
      either way; check `ls -la /dev/serial0` and whether `hciuart`/Bluetooth is active.
- [ ] GPS-disciplined time service exposing "best UTC estimate + confidence" over MQTT/REST for the
      celestial app's chronometer tab (drift tracking/logging when GPS is lost is mostly covered by
      chrony's own drift file + `rtcsync`, but the navcomputer needs its own confidence/age signal
      for the UI, not just raw chrony state)

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

- ~~Exact UPS fuel-gauge chip + I2C address~~ **Resolved**: PiSugar 3, I2C `0x57`
- ~~Confirm Pi is Pi 4 (has onboard WiFi — confirmed) and whether a DS3231 (or similar) RTC will be added~~ **Resolved**: Pi 4, DS3231 RTC added, USB WiFi dongle added, GPS PPS wired to GPIO6
- Preferred speed-source priority/fallback order for DR (GPS → log sensor → manual?)
- HP48GX cable: wired serial vs IR dongle
- Which USB WiFi adapter model (for `hostapd` AP-mode driver compatibility) and which radio
  (onboard vs dongle) will run the hotspot vs any uplink
  - **Found**: dongle enumerates via `lsusb` as "Aicsemi Aic MSC" (Mass Storage Class) — typical
    of AIC8800-chipset dongles stuck in CD-ROM/installer mode; no `wlan1`-style interface appears
    yet. Needs a mode-switch + the AIC8800 Linux driver before it's usable for the hotspot.
- Confirm GPS module actually drives a PPS output pin wired to GPIO6, and which UART (`uart3` vs
  the primary) it's connected to — needed before implementing PPS-based time discipline in Phase 3
