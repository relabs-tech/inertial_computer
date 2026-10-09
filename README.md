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
- Standalone HMC5983 magnetometer (via I2C)
  - ✅ **New HMC5983 driver** with comprehensive configuration support
  - ✅ Configurable sample rates, gain, averaging, and modes (continuous/single)
  - ✅ Raw and scaled output (µT × 10 for consistency with internal mag convention)
  - ✅ Independent MQTT topic publishing: `inertial/mag/hmc`
  - ✅ Configuration-driven I2C address, bus, and sensor parameters
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
  - ✅ Real-time dashboard with all sensor data (IMU, BMP, GPS, HMC5983)
  - ✅ Satellite sky plot (polar chart showing satellite positions)
  - ✅ Satellite signal strength bar chart
  - ✅ Weather widget with external API integration
  - ✅ **Interactive calibration UI** with 3D visualization and guided workflows
- CLI calibration tool
  - ✅ **Console-based calibration** for gyroscope, accelerometer, magnetometer (left/right IMUs)
  - ✅ Guided step-by-step process with confidence scoring
  - ✅ JSON output with timestamped calibration files
- Display consumer (SSD1306 OLED displays)
  - ✅ **Dual display support** with configurable I2C addresses
  - ✅ **Configurable content** per display (raw IMU, orientation, GPS, HMC5983 mag)
  - ✅ Real-time updates at configurable intervals
  - ✅ Support for: `imu_raw_left`, `imu_raw_right`, `orientation_left`, `orientation_right`, `gps`, `hmc5983`
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
- ✅ **Standalone HMC5983 magnetometer driver** with comprehensive configuration
  - Configurable I2C bus and address via config file
  - Selectable output data rates (ODR): 3Hz, 7.5Hz, 15Hz, 30Hz, 75Hz (configurable)
  - Gain selection (0-7) for different measurement ranges
  - Sample averaging (1, 2, 4, 8) for noise reduction
  - Continuous and single-shot measurement modes
  - Raw and scaled output (µT × 10) matching internal magnetometer conventions
  - Dedicated MQTT topic: `inertial/mag/hmc` with magnitude and timestamp
- ✅ GPS module with comprehensive NMEA parsing (RMC, GGA, GSA, VTG, GSV)
- ✅ Satellite tracking with signal strength visualization
- ✅ MQTT-based message bus architecture with topic-based data distribution
- ✅ Console subscriber for real-time monitoring
- ✅ Web UI with REST API for data access
- ✅ Satellite visualizations (sky plot and signal strength bar chart)
- ✅ Weather integration with met.no API based on GPS location
- ✅ Sea level pressure calculation from local BMP sensors
- ✅ Magnetometer driver integration with dedicated topics (left/right IMUs + HMC5983)
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
- **HMC5983 Magnetometer Driver** (2026-01-08): New standalone magnetometer support
  - Complete HMC5983/HMC5883L driver in periph.io devices fork
  - Configuration-driven setup via `inertial_config.txt`
  - Independent I2C producer (`cmd/hmc5983_producer`) with MQTT publishing
  - Configurable ODR, gain, averaging, and measurement modes
  - Raw register values (X,Z,Y hardware order) with automatic reordering to X,Y,Z
  - Scaled output in µT × 10 for consistency with project conventions
  - Magnitude calculation and RFC3339 timestamping in MQTT payload
  - Can operate alongside dual IMU setup for multi-sensor magnetometer fusion
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

## Celestial Navigation App (vendored dependency)

[`celestial/`](celestial) is a separate web app (Three.js sight-reduction tool), vendored as a **nested git repository** with its own remote (`git@github.com:Daniel-relabs/celestial.git`), rather than a submodule. It's served on the Pi by `cmd/celestial` (see [ARCHITECTURE.md](ARCHITECTURE.md) §6.6) and linked from the main dashboard.

This repo has one local-only modification (a "Use inertial-computer GPS fix" button in `celestial/index.html`) that should never be pushed upstream to the `celestial` repo. To keep that change isolated while still being able to pull new upstream commits:

**One-time setup** — move the local edit onto a branch that's never pushed:
```bash
cd celestial
git checkout -b local/inertial-dashboard
git add -A
git commit -m "Add GPS-fix button for inertial-computer dashboard"
```
`main` now mirrors upstream exactly; the dashboard-specific change lives only on `local/inertial-dashboard`. Since that branch has no tracking upstream configured, a bare `git push` won't touch it — only push `main` explicitly.

**Pulling upstream updates later:**
```bash
cd celestial
git checkout main
git pull origin main
git checkout local/inertial-dashboard
git rebase main
```
Resolve any conflicts normally (only possible if upstream touches the same lines you modified), then make sure `celestial/` is checked out on `local/inertial-dashboard` when deploying — that's the branch with the working dashboard integration.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

**Copyright © 2026 Daniel Alarcon Rubio / Relabs Tech**

## Author

**Daniel Alarcon Rubio**
- Organization: Relabs Tech
- GitHub: [@relabs-tech](https://github.com/relabs-tech)

---
