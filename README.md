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
- ✅ Magnetometer driver integration with dedicated topics for left and right sensors
- ✅ Environmental sensors (BMP280/BMP388 temperature and pressure via SPI)
- ✅ Configuration system with centralized `inertial_config.txt`
- ✅ IMU manager with singleton pattern for persistent hardware access

**In Progress**:
- ⚠️ Magnetometer calibration (hard-iron and soft-iron correction)
- ⚠️ Sensor fusion (integrating gyro and mag into yaw calculation)

**Recent Changes**:
- **IMU Refactoring**: Implemented singleton IMUManager pattern for persistent sensor access
- **BMP Integration**: Added bmxx80 driver support for real temperature and pressure readings with multiple units (Pa, mbar, hPa)
- **Configuration System**: Externalized all hardcoded values to `inertial_config.txt` (MQTT broker, topics, SPI devices, GPIO pins, timing intervals)
- **Right Magnetometer**: Added dedicated MQTT topic `inertial/mag/right` for right sensor magnetometer data
- **Configurable Logging**: Console output now configurable with `CONSOLE_LOG_INTERVAL` setting

See [TODO.md](TODO.md) for detailed task list and [ARCHITECTURE.md](ARCHITECTURE.md) for system design.

---

## Configuration

All system settings are centralized in `inertial_config.txt` at the project root. This includes:

- **MQTT broker address and client IDs**
- **MQTT topic names** for all data streams
- **IMU hardware settings** (SPI devices and CS pins)
- **GPS serial port** and baud rate
- **Timing intervals** (IMU sample rate)
- **Web server port**

To customize your setup, edit `inertial_config.txt` before running any program. The configuration file uses a simple `KEY=VALUE` format with comments starting with `#`.

Example configuration snippet:
```
MQTT_BROKER=tcp://localhost:1883
IMU_LEFT_SPI_DEVICE=/dev/spidev6.0
IMU_LEFT_CS_PIN=18
GPS_SERIAL_PORT=/dev/serial0
IMU_SAMPLE_INTERVAL=100
```

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
