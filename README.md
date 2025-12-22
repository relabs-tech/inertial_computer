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
  - ✅ Full NMEA sentence parsing (RMC, GGA, GSA, VTG, GSV)
  - ✅ Satellite tracking with elevation, azimuth, and signal strength
  - ✅ Multiple GPS data topics (position, velocity, quality, satellites)
- Fused orientation (roll / pitch / yaw)
- Weather data from met.no API
  - ✅ Temperature, pressure, humidity, and conditions based on GPS location
  - ✅ Local sea level pressure calculation from BMP sensors

### Data consumers
- Console MQTT subscriber
- Web server + browser UI
  - ✅ Real-time dashboard with all sensor data
  - ✅ Satellite sky plot (polar chart showing satellite positions)
  - ✅ Satellite signal strength bar chart
  - ✅ Weather widget with external API integration

---

## Current Status

**Working**:
- ✅ Dual IMU setup (left and right MPU9250) reading accel, gyro, and magnetometer
- ✅ GPS module with comprehensive NMEA parsing (RMC, GGA, GSA, VTG, GSV)
- ✅ Satellite tracking with signal strength visualization
- ✅ MQTT-based message bus architecture with topic-based data distribution
- ✅ Console subscriber for real-time monitoring
- ✅ Web UI with REST API for data access
- ✅ Satellite visualizations (sky plot and signal strength bar chart)
- ✅ Weather integration with met.no API based on GPS location
- ✅ Sea level pressure calculation from local BMP sensors
- ✅ Magnetometer driver integration with dedicated topics for left and right sensors
- ✅ Environmental sensors (BMP280/BMP388 temperature and pressure via SPI)
- ✅ Configuration system with centralized `inertial_config.txt`
- ✅ IMU manager with singleton pattern for persistent hardware access
- ✅ Optimized dashboard layout for single-screen viewing

**In Progress**:
- ⚠️ Magnetometer calibration (hard-iron and soft-iron correction)
- ⚠️ Sensor fusion (integrating gyro and mag into yaw calculation)

**Recent Changes**:
- **Enhanced GPS**: Full NMEA support with RMC, GGA, GSA, VTG, and GSV sentence parsing
- **Satellite Tracking**: Real-time satellite visibility with elevation, azimuth, and signal strength (SNR)
- **GPS Topic Split**: Separate MQTT topics for position, velocity, quality, and satellites data
- **Satellite Visualizations**: Added sky plot (polar chart) and signal strength bar chart to web UI
- **Weather Integration**: Met.no API integration providing temperature, pressure, humidity, and conditions based on GPS location
- **Sea Level Pressure**: Automatic SLP calculation from local BMP sensors corrected for GPS altitude
- **Dashboard Optimization**: Compact layout designed to fit all widgets on a single screen
- **Pressure Units**: Changed from Pa to hPa for better readability
- **Weather Caching**: Configurable API fetch interval (default 5 minutes) to respect rate limits
- **Project Cleanup**: Renamed `cmd/producer` to `cmd/imu_producer` for clarity
- **IMU Refactoring**: Implemented singleton IMUManager pattern for persistent sensor access
- **BMP Integration**: Added bmxx80 driver support for real temperature and pressure readings
- **Configuration System**: Externalized all hardcoded values to `inertial_config.txt`

See [TODO.md](TODO.md) for detailed task list and [ARCHITECTURE.md](ARCHITECTURE.md) for system design.

---

## Configuration

All system settings are centralized in `inertial_config.txt` at the project root. This includes:

- **MQTT broker address and client IDs**
- **MQTT topic names** for all data streams (including GPS subtopics)
- **IMU hardware settings** (SPI devices and CS pins)
- **BMP hardware settings** (SPI devices)
- **GPS serial port** and baud rate
- **Timing intervals** (IMU sample rate, console logging)
- **Web server port**
- **Weather API update interval** (minutes between met.no API calls)

To customize your setup, edit `inertial_config.txt` before running any program. The configuration file uses a simple `KEY=VALUE` format with comments starting with `#`.

Example configuration snippet:
```
MQTT_BROKER=tcp://localhost:1883
IMU_LEFT_SPI_DEVICE=/dev/spidev6.0
IMU_LEFT_CS_PIN=18
GPS_SERIAL_PORT=/dev/serial0
IMU_SAMPLE_INTERVAL=100
WEATHER_UPDATE_INTERVAL_MINUTES=5
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
