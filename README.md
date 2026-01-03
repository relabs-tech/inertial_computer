# Inertial Computer

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

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
  - ✅ **Interactive calibration UI** with 3D visualization and guided workflows
- CLI calibration tool
  - ✅ **Console-based calibration** for gyroscope, accelerometer, magnetometer
  - ✅ Guided step-by-step process with confidence scoring
  - ✅ JSON output with timestamped calibration files
- Display consumer (SSD1306 OLED displays)
  - ✅ **Dual display support** with configurable I2C addresses
  - ✅ **Configurable content** per display (raw IMU, orientation, GPS)
  - ✅ Real-time updates at configurable intervals
  - ✅ Support for: `imu_raw_left`, `imu_raw_right`, `orientation_left`, `orientation_right`, `gps`
- Register debugger (MPU9250 hardware debugging)
  - ✅ **Direct register access** to all 128 MPU9250 registers
  - ✅ **Bitfield manipulation** with toggle switches for configuration registers
  - ✅ **Live sensor monitoring** during register modifications
  - ✅ **SPI speed control** for debugging timing issues
  - ✅ **Configuration export/import** with JSON persistence

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
- ✅ Display consumer with dual SSD1306 OLED support and configurable content
- ✅ **Configurable IMU sample rates** with DLPF, sample rate divider, and accel DLPF settings

**In Progress**:
- ⚠️ Magnetometer calibration application to sensor readings
- ⚠️ Sensor fusion (integrating gyro and mag into yaw calculation)
- ⚠️ Apply calibration coefficients in producers

**Recent Changes**:
- **GPS/GLONASS Separation** (2025-01-02): Fixed satellite data display issue
  - Separated GPGSV (GPS) and GLGSV (GLONASS) constellation processing
  - Added raw NMEA logging for debugging (`[GPS-RAW]` prefix)
  - Separate MQTT topics: `inertial/gps/satellites` and `inertial/glonass/satellites`
  - Web UI distinguishes GPS (circles) vs GLONASS (squares) in visualizations
  - Resolved data pollution: lack of GLONASS satellites no longer affects GPS display
  - Topic-specific payloads prevent cross-constellation data contamination
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
- **BMP Configuration**: Added comprehensive sensor configuration (oversampling, IIR filter, standby time, mode) for both left and right BMP sensors
- **IMU Sensor Ranges**: Configurable accelerometer (±2g to ±16g) and gyroscope (±250°/s to ±2000°/s) ranges
- **Web Calibration UI**: Interactive 3D-guided calibration interface with real-time visualization (Three.js)
- **CLI Calibration Tool**: Console-based alternative with step-by-step guided workflows
- **Calibration Output**: Timestamped JSON files with bias, scale factors, and confidence metrics
- **Register Debug Tool** (2025-01-03): WebSocket-based MPU9250 hardware debugging
  - Comprehensive register table with read/write operations
  - Bitfield manipulation with toggle switches and real-time preview
  - Live sensor data monitoring during register modifications
  - SPI speed control for debugging timing problems
  - Configuration export/import with JSON persistence
  - Safety features: read-only indicators, bitfield validation, confirmation dialogs
  - Runs on port 8081, accessible from main dashboard
- **Custom periph.io Fork Enhancements** (2025-01-03): Enhanced drivers for magnetometer and OLED displays
  - **MPU9250 magnetometer (AK8963)**: Internal I2C master support with calibration data structures
  - **SSD1306 display driver**: Dual I2C address support, optimized VerticalLSB pixel format, differential updates
  - Enables efficient multi-sensor operation and real-time OLED rendering on embedded I2C buses
  - See [QUICKSTART.md](QUICKSTART.md) Step 0 for fork installation and complete feature list

See [TODO.md](TODO.md) for detailed task list, [ARCHITECTURE.md](ARCHITECTURE.md) for system design, [CALIBRATION_UI.md](CALIBRATION_UI.md) for calibration UI details, and [QUICKSTART.md](QUICKSTART.md) for setup instructions.

---

## Configuration

All system settings are centralized in `inertial_config.txt` at the project root. This includes:

- **MQTT broker address and client IDs**
- **MQTT topic names** for all data streams (including GPS subtopics)
- **IMU hardware settings** (SPI devices and CS pins)
- **IMU sensor ranges** (accelerometer: ±2g/±4g/±8g/±16g, gyroscope: ±250°/s to ±2000°/s)
- **BMP hardware settings** (SPI devices)
- **BMP sensor configuration** (oversampling, filter, standby time for both left and right sensors)
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
IMU_ACCEL_RANGE=2
IMU_GYRO_RANGE=1
BMP_LEFT_SPI_DEVICE=/dev/spidev6.1
BMP_LEFT_PRESSURE_OSR=5
BMP_LEFT_TEMP_OSR=2
BMP_LEFT_IIR_FILTER=3
BMP_LEFT_STANDBY_TIME=1
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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

**Copyright © 2026 Daniel Alarcon Rubio / Relabs Tech**

## Author

**Daniel Alarcon Rubio**
- Organization: Relabs Tech
- GitHub: [@relabs-tech](https://github.com/relabs-tech)

---
