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
- **[NEW]** Separated raw sensor reads from pose computation (branch: `refactor/separate-raw-reads-from-pose`)
  - `IMURawReader` interface for hardware reads
  - Pure functions `AccelToPose()`, `ComputePoseFromAccel()` in orientation package
  - Producer refactored to call `ReadRaw()` then compute pose

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
- ⚠️ Implement gyro reads (TODO: GetGyro* methods)
- ⚠️ Configure internal I2C master for AK8963 magnetometer
- ⚠️ Read magnetometer from EXT_SENS_DATA registers

### Right IMU
- ❌ SPI wiring not yet done
- ❌ Implement right IMU driver (same as left, different SPI/CS)
- ❌ Implement ReadRightIMURaw stub

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
- ⚠️ Currently publishes zeros for gyro/mag (driver TODO)
- ⚠️ Currently publishes zeros for BMP (driver TODO)
- Ready for: gyro/mag fusion, multi-sensor fusion

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

1. **Merge refactor branch** — raw reads separated from pose computation
2. **Gyro driver integration** — read actual gyroscope values in `ReadRaw()`
3. **Gyro integration function** — integrate angular velocity to get yaw estimate
4. **Magnetometer driver integration** — read from EXT_SENS_DATA (internal I2C slave)
5. **Magnetometer correction function** — compute yaw from mag + soft-iron calibration
6. **Complementary filter** — blend accel/gyro/mag for robust orientation
7. **Right IMU driver** — duplicate left IMU logic with different SPI/CS
8. **BMP environmental sensor driver** — I2C temperature/pressure reads
9. **Dual-IMU fusion** — cross-validate and combine left/right readings
10. **Optional: Kalman filter** — advanced fusion for production use

---
