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
- ❌ Initialize BMP sensors on I2C
- ❌ Read temperature and pressure
- ❌ Implement ReadLeftEnv / ReadRightEnv stubs
- Note: currently return zero values

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

### Inertial Producer (`cmd/producer`)
- ✅ Refactored to call `ReadRaw()` and `AccelToPose()` separately
- ✅ Mock mode still works (can switch via `useMock` flag)
- ✅ Left IMU gyro values are now read and published
- ✅ Left and right IMU magnetometer values are now read and published
- ✅ Producer logs pose, accel, gyro, and mag (with magnitude) each 100ms tick
- ✅ Test/debug MQTT topic `inertial/mag/left` publishes magnetometer data with norm
- ⚠️ Currently publishes zeros for BMP (driver TODO)
- ⚠️ Magnetometer calibration not yet applied (hard/soft-iron correction TODO)
- ⚠️ Magnetometer not yet integrated into yaw calculation
- Ready for: magnetometer calibration, gyro/mag fusion, multi-sensor fusion

### GPS Producer
- ✅ Functional (reads NMEA, publishes GPS fixes)

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
- To add: HARDWARE.md, CALIBRATION.md

---

## 8. Next steps (priority order)

1. **Merge refactor branch** — raw reads separated from pose computation ✅ done
2. **Gyro driver integration** — read actual gyroscope values in `ReadRaw()` ✅ done
3. **Right IMU driver** — duplicate left IMU logic with different SPI/CS ✅ done
4. **Magnetometer driver integration** — read from EXT_SENS_DATA (internal I2C slave) ✅ done (test code active)
5. **Magnetometer calibration** — implement hard-iron and soft-iron correction
6. **Gyro integration function** — integrate angular velocity to get yaw estimate
7. **Magnetometer correction function** — compute yaw from mag + calibration applied
8. **Complementary filter** — blend accel/gyro/mag for robust orientation
9. **BMP environmental sensor driver** — I2C temperature/pressure reads
10. **Dual-IMU fusion** — cross-validate and combine left/right readings
11. **Optional: Kalman filter** — advanced fusion for production use

---

## Additional high-priority tasks (requested)

1. **Fuse Left and Right IMUs for Pose** — implement fusion algorithm to combine left and right IMU readings into a single, robust `Pose` output. Start with simple averaging or weighted fusion, then move to complementary filter or EKF as needed.

2. **Wire up BMPxx80 sensors** — add drivers for BMP280/BMP388, initialize on I2C, and implement `ReadLeftEnv()` / `ReadRightEnv()` to publish temperature and pressure.

3. **Calibration Application (Web UI)** — separate standalone calibration tool with its own code and drivers
   - Web UI for interactive calibration procedures
   - Dedicated command (`cmd/calibration` or similar)
   - Own driver instances for left/right IMUs and BMPs (independent from main producer)
   - Calibration functions:
     - **Accelerometer calibration**: capture bias, scale factors for left/right
     - **Gyroscope calibration**: capture drift/bias for left/right
     - **Magnetometer calibration**: hard-iron and soft-iron correction for left/right
     - **BMP calibration**: temperature coefficient adjustment if needed
   - Store calibration parameters persistently (config file or database)
   - Apply calibration coefficients when main producer reads sensors

4. **Calibrate accelerometers and gyros** — add calibration routines (bias estimation, scale factors) and tooling/documentation to persist calibration parameters.

5. **Add the magnetometers (AK8963)** — enable MPU9250 internal I2C master, read AK8963 via EXT_SENS_DATA, add soft-iron / hard-iron calibration, and expose magnetometer data for yaw fusion.

6. **Incorporate two SSD1306 displays** — wire and initialize two SSD1306 I2C/OLED displays, add lightweight UI showing key telemetry (pose, imu values, connection status) and expose a simple API to update display content.

7. **Add CLI options to ./cmd/xxx/main.go apps** — use Go's flag package to add configurable options like MQTT broker URL, IMU selection (mock/real), debug levels, serial ports, and web server ports. Update RunXXX functions to accept config structs. Low effort (1-2 hours total for all 5 apps).

---
