# Inertial Computer

Developer-oriented inertial sensing platform built in Go, designed around a message-bus (MQTT) architecture. The system reads multiple hardware sensors (IMUs, magnetometers, environmental sensors, GPS), publishes all data streams via MQTT, and exposes multiple consumers such as a console viewer and a web-based UI.

The core goals are:
- clean separation between hardware access, data transport, and presentation
- ability to add sensors or consumers without restructuring the system
- support for raw data inspection and higher-level fused outputs

---

## What the system does

### Sensors / data sources
- Left IMU (accelerometer + gyroscope + magnetometer)
  - MPU9250 with AK8963 magnetometer via internal I2C
  - ✅ Accelerometer and gyroscope fully operational
  - ✅ Magnetometer initialized and reading (test/debug mode)
- Right IMU (accelerometer + gyroscope + magnetometer)
  - MPU9250 with AK8963 magnetometer via internal I2C
  - ✅ Accelerometer and gyroscope fully operational
  - ✅ Magnetometer initialized and reading (test/debug mode)
- Left environmental sensor (BMP: temperature + pressure)
- Right environmental sensor (BMP: temperature + pressure)
- GPS (NMEA-based)
- Fused orientation (roll / pitch / yaw)

### Data consumers
- Console MQTT subscriber
- Web server + browser UI

---

## Current Status

**Working**:
- ✅ Dual IMU setup (left and right MPU9250) reading accel, gyro, and magnetometer
- ✅ GPS module (NMEA parsing and publishing)
- ✅ MQTT-based message bus architecture
- ✅ Console subscriber for real-time monitoring
- ✅ Web UI with REST API for data access
- ✅ Magnetometer driver integration complete (test/debug mode active)

**In Progress**:
- ⚠️ Magnetometer calibration (hard-iron and soft-iron correction)
- ⚠️ Sensor fusion (integrating gyro and mag into yaw calculation)
- ⚠️ Environmental sensors (BMP temperature/pressure drivers)

**Recent Changes**:
- Magnetometer (AK8963) driver integrated via internal I2C on MPU9250
- Test MQTT topic `inertial/mag/left` publishing magnetometer data with field magnitude
- Local fork of `periph.io/x/devices` for magnetometer support

See [TODO.md](TODO.md) for detailed task list and [ARCHITECTURE.md](ARCHITECTURE.md) for system design.

---

## Hardware Connection

The Raspberry Pi requires specific hardware interfaces enabled in `/boot/firmware/config.txt`:

```plaintext
# Uncomment some or all of these to enable the optional hardware interfaces
dtparam=i2c_arm=on
#dtparam=i2s=on
dtparam=spi=on

dtoverlay=spi6-2cs,cs0_pin=18,cs1_pin=27
dtoverlay=spi0-2cs,cs0_pin=8,cs1_pin=7
```

- **I2C**: Enabled via `dtparam=i2c_arm=on` for MPU9250 IMU communication
- **SPI**: Enabled via `dtparam=spi=on` for additional sensor interfaces
- **SPI6**: Configured with CS pins 18 and 27
- **SPI0**: Configured with CS pins 8 and 7

---
