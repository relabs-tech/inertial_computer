# Inertial Computer – Consolidated TODO & Project State

This document consolidates everything discussed so far into a single place.
It is intended as a restart guide after a pause, capturing architecture decisions,
current state, known-good references, and all remaining work.

---

## 1. Current state

### Completed
- Repository and Go module structure stabilized
- MQTT-based architecture implemented
- Console MQTT subscriber working
- Web server + REST API working
- Web UI readable and functional
- GPS ingestion and publication working
- Core domain models defined
- README.md and ARCHITECTURE.md written
- **[DONE]** Separated raw sensor reads from pose computation (branch: `refactor/separate-raw-reads-from-pose`)
  - `IMURawReader` interface for hardware reads
  - Pure functions `AccelToPose()`, `ComputePoseFromAccel()` in orientation package
  - Producer refactored to call `ReadRaw()` then compute pose
- **[DONE]** Right IMU integration (branch: `feature/imu-right`)
  - Right IMU accessible via SPI0, CS on GPIO 8
  - Accel, gyro reads working
- **[DONE]** Gyroscope integration (commit: 34a11cf)
  - Left and right IMU gyro values read via `GetRotationX/Y/Z()`
  - Published in IMURaw structs
- **[NEW]** Magnetometer driver integration (branch: `Mag_Add`, commits: 139a91d, f5f3cb3)
  - Left and right IMU magnetometer (AK8963) initialized via internal I2C
  - `InitMag()` and `ReadMag()` implemented in driver
  - Magnetometer reads working; values published in IMURaw structs
  - Test/debug MQTT topic `inertial/mag/left` publishing mag data with field magnitude
  - Producer logs include magnetometer readings and |B| magnitude
  - Local fork of `periph.io/x/devices` integrated via replace directive

### Not completed
- Real IMU (MPU9250) wiring (accel reads work, gyro/mag TODOs remain)
- Magnetometer (AK8963) integration
- BMP environmental sensors
- Proper sensor fusion

---

## 2. Architecture recap

Data flow:
Sensors → Producers → MQTT → Consumers

Key constraint:
Pi → SPI → MPU9250 → internal I2C → AK8963

The Pi never talks directly to the magnetometer.

---

## 3. Sensor layer (internal/sensors)

### Left IMU
- ✅ Access MPU9250 via SPI (working)
- ✅ Read accel via `imuSource.ReadRaw()` (working)
- ✅ Read gyroscope (rotation) values via `GetRotationX/Y/Z` (implemented)
- ✅ Configure internal I2C master for AK8963 magnetometer (complete)
- ✅ Read magnetometer from EXT_SENS_DATA registers (working with test code)
- ⚠️ Magnetometer calibration (hard-iron and soft-iron correction TODO)
- ⚠️ Integration of magnetometer into yaw calculation (fusion TODO)

### Right IMU
- ✅ Access MPU9250 via SPI (working, wired and tested)
- ✅ Read accel via `ReadRightIMURaw()` (working)
- ✅ Read gyroscope (rotation) values via `GetRotationX/Y/Z()` (implemented)
- ✅ Configure internal I2C master for AK8963 magnetometer (complete)
- ✅ Read magnetometer from EXT_SENS_DATA registers (working with test code)
- ⚠️ Magnetometer calibration (hard-iron and soft-iron correction TODO)
- ⚠️ Integration of magnetometer into yaw calculation (fusion TODO)

### Environmental sensors (BMP)
- ✅ Initialize BMP sensors on SPI
- ✅ Read temperature and pressure
- ✅ Implement ReadLeftEnv / ReadRightEnv
- ✅ Configure oversampling, IIR filter, standby time, and operating mode
- ✅ Independent configuration for left and right BMP sensors via inertial_config.txt
- Note: Fully operational with configurable parameters

---

## 4. Orientation & pose computation

### Current (refactor branch)
- ✅ `orientation.AccelToPose()` — pure function, converts accel to roll/pitch
- ✅ Separated from hardware reads (`ReadRaw()` is independent)
- ⚠️ Yaw hardcoded to 0 (ready for magnetometer fusion)

### Next phases
- Phase 1: Read actual gyro data, integrate over time for stable yaw
- Phase 2: Add magnetometer correction via EXT_SENS_DATA
- Phase 3: Implement complementary filter (accel + gyro + mag)
- Phase 4: Dual-IMU cross-validation and fusion
- Phase 5: Optional EKF for advanced scenarios
---

## 5. Producers

### IMU Producer (`cmd/imu_producer`, renamed from `cmd/producer`)
- ✅ Refactored to call `ReadRaw()` and `AccelToPose()` separately
- ✅ Mock mode still works (can switch via `useMock` flag)
- ✅ Left IMU gyro values are now read and published
- ✅ Left and right IMU magnetometer values are now read and published
- ✅ Producer logs pose, accel, gyro, and mag (with magnitude) each 100ms tick
- ✅ Test/debug MQTT topic `inertial/mag/left` publishes magnetometer data with norm
- ✅ Right magnetometer MQTT topic `inertial/mag/right` publishes right sensor data
- ✅ BMP sensors fully integrated with bmxx80 driver (temperature and pressure in Pa, mbar, hPa)
- ✅ Configuration system implemented - all hardcoded values externalized to `inertial_config.txt`
- ✅ IMU manager singleton pattern - persistent hardware access without re-initialization
- ✅ Configurable console logging interval via `CONSOLE_LOG_INTERVAL` setting
- ⚠️ Magnetometer calibration not yet applied (hard/soft-iron correction TODO)
- ⚠️ Magnetometer not yet integrated into yaw calculation
- Ready for: magnetometer calibration, gyro/mag fusion, multi-sensor fusion

### GPS Producer (`cmd/gps_producer`)
- ✅ Comprehensive NMEA parsing (RMC, GGA, GSA, VTG, GSV sentences)
- ✅ Multi-topic publishing: position, velocity, quality, satellites, legacy
- ✅ Satellite tracking with elevation, azimuth, and SNR
- ✅ GSV sentence accumulation logic (handles multi-sentence satellite messages)
- ✅ Configurable MQTT topics via `inertial_config.txt`

---

## 6. Consumers

### Console MQTT
- Functional
- Optional formatting improvements

### Web UI
- Functional
- Optional stream health indicators
- Optional 3D visualization

---

## 7. Documentation

- README.md ✔
- ARCHITECTURE.md ✔
- CALIBRATION_UI.md ✔
- QUICKSTART.md ✔
- To add: HARDWARE.md (detailed wiring/pinout guide)

---

## 8. Next steps (priority order)

1. **Merge refactor branch** — raw reads separated from pose computation ✅ done
2. **Gyro driver integration** — read actual gyroscope values in `ReadRaw()` ✅ done
3. **Right IMU driver** — duplicate left IMU logic with different SPI/CS ✅ done
4. **Magnetometer driver integration** — read from EXT_SENS_DATA (internal I2C slave) ✅ done
5. **Right magnetometer topic** — add dedicated publishing for right sensor ✅ done
6. **BMP environmental sensor driver** — SPI temperature/pressure reads with bmxx80 ✅ done
7. **Configuration system** — externalize all hardcoded values to config file ✅ done
8. **IMU manager singleton** — persistent hardware access pattern ✅ done
9. **Calibration tools** ✅ COMPLETED — web UI and CLI tools for gyro/accel/mag calibration
10. **Apply calibration in producers** — load calibration JSON and apply corrections to sensor reads
11. **Gyro integration function** — integrate angular velocity to get yaw estimate
12. **Magnetometer correction function** — compute yaw from mag + calibration applied
13. **Complementary filter** — blend accel/gyro/mag for robust orientation
14. **Dual-IMU fusion** — cross-validate and combine left/right readings
15. **Optional: Kalman filter** — advanced fusion for production use

---

## Additional high-priority tasks (requested)

1. **Fuse Left and Right IMUs for Pose** — implement fusion algorithm to combine left and right IMU readings into a single, robust `Pose` output. Start with simple averaging or weighted fusion, then move to complementary filter or EKF as needed.

2. **Wire up BMPxx80 sensors** ✅ COMPLETED
   - ✅ Added bmxx80 drivers for BMP280/BMP388 via SPI
   - ✅ Implemented `ReadLeftEnv()` / `ReadRightEnv()` with temperature and pressure
   - ✅ Pressure output in multiple units: Pa, mbar, and hPa
   - ✅ Uses singleton pattern with sync.Once for initialization
   - ✅ Configuration-driven SPI device paths
   - ✅ **NEW**: Configurable sensor parameters (oversampling, IIR filter, standby time, mode)
   - ✅ **NEW**: Independent configuration for left and right sensors via `inertial_config.txt`
   - ✅ **NEW**: Default settings optimized for accuracy (16x pressure, 2x temp, F8 filter, 62.5ms standby)

3. **Calibration Application** ✅ COMPLETED
   - ✅ Web UI with interactive 3D-guided calibration (Three.js visualization)
   - ✅ CLI tool at `cmd/calibration` for console-based calibration
   - ✅ WebSocket-based real-time communication for web UI
   - ✅ Dedicated calibration handler with state machine
   - ✅ Calibration functions implemented:
     - **Gyroscope calibration**: Static bias + per-axis dynamic refinement
     - **Accelerometer calibration**: 6-point orientation capture with bias and scale
     - **Magnetometer calibration**: Min/max ellipsoid for hard-iron offset and soft-iron diagonal scale
   - ✅ JSON output with timestamped calibration files
   - ✅ Confidence scoring for each sensor type
   - ⚠️ Apply calibration coefficients in producers (TODO)
   - ⚠️ Persistent calibration profile management (TODO)

4. **Calibrate accelerometers and gyros** ✅ TOOLS COMPLETED
   - ✅ Calibration routines implemented in both web UI and CLI tool
   - ✅ Bias estimation and scale factors calculated
   - ✅ JSON persistence with timestamped output files
   - ⚠️ Integration into producer pipeline (apply corrections to sensor reads) TODO

5. **Add the magnetometers (AK8963)** ✅ COMPLETED
   - ✅ Enabled MPU9250 internal I2C master
   - ✅ Read AK8963 via EXT_SENS_DATA registers
   - ✅ Exposed magnetometer data for both left and right sensors
   - ✅ Added dedicated MQTT topics for left and right magnetometers
   - ⚠️ Soft-iron / hard-iron calibration TODO
   - ⚠️ Integration into yaw fusion TODO

6. **Incorporate two SSD1306 displays** — wire and initialize two SSD1306 I2C/OLED displays, add lightweight UI showing key telemetry (pose, imu values, connection status) and expose a simple API to update display content.

7. **Add CLI options to ./cmd/xxx/main.go apps** ✅ PARTIALLY COMPLETED
   - ✅ Configuration system implemented via `inertial_config.txt`
   - ✅ All apps read config at startup (MQTT broker, ports, hardware settings)
   - ⚠️ Runtime CLI flags not yet implemented (could add flag overrides for config values)

---
