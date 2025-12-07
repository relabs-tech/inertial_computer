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

### Not completed
- Real IMU (MPU9250) wiring
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
- Access MPU9250 via SPI
- Implement SPI register read/write if missing
- Configure internal I2C master for AK8963
- Read accel, gyro, magnetometer
- Implement ReadLeftIMURaw

### Right IMU
- Same as left IMU with different SPI/CS
- Implement ReadRightIMURaw

### Environmental sensors
- Initialize BMP sensors on I2C
- Read temperature and pressure
- Implement ReadLeftEnv / ReadRightEnv

---

## 4. Orientation logic

- Current: accel-based roll/pitch
- Next: magnetometer yaw from EXT_SENS_DATA
- Medium-term: complementary filter
- Long-term: dual-IMU fusion

---

## 5. Producers

- Replace mock data with real sensor reads
- Publish IMU, BMP, and fused pose topics

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

## 8. Resume order

1. Left IMU (accel/gyro)
2. Verify roll/pitch
3. Add magnetometer
4. Verify yaw
5. Add right IMU
6. Add BMP sensors
7. Switch from mock data
8. Start fusion

---
