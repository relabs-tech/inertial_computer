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
- **Display consumer** ✅ Dual SSD1306 OLED support with configurable content
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
- **[DONE]** GPS/GLONASS constellation separation (2025-01-02)
  - Raw NMEA logging with `[GPS-RAW]` prefix for debugging
  - Separate processing of GPGSV (GPS) and GLGSV (GLONASS) satellite data
  - Data structures updated: `gps.Fix` and `gps.SatellitesInView` with separate GPS/GLONASS fields
  - Added `TOPIC_GLONASS_SATELLITES` configuration for separate MQTT publishing
  - Web UI visualization distinguishes GPS (circles) vs GLONASS (squares)
  - Fixed satellite data display issue: lack of GLONASS data no longer pollutes GPS display
  - Topic-specific anonymous structs prevent cross-constellation data contamination in MQTT payloads

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
   - Load most recent calibration file for selected IMU (left/right)
   - Apply gyro bias correction: `corrected = raw - bias`
   - Apply accel bias and scale: `corrected = (raw - bias) * scale`
   - Apply mag hard-iron offset and soft-iron scale: `corrected = (raw - offset) * scale`
   - Configuration option to specify calibration file path or auto-detect latest
   - Fallback to uncorrected data if calibration file not found
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
   - ⚠️ **Magnetometer test visualization** (TODO)
     - Add "Test Magnetometer" button to calibration screen
     - Launch 3D scatter plot widget (Three.js) showing real-time mag data points (Mx, My, Mz)
     - Continuously plot points as user moves device to visualize magnetic field sphere/ellipsoid
     - Include "Back to Calibration" button to return to main calibration screen
     - Useful for verifying calibration quality and detecting magnetic distortions

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

6. **Incorporate two SSD1306 displays** ✅ COMPLETED
   - ✅ Dual SSD1306 128x64 OLED displays via I2C
   - ✅ Configurable I2C addresses (default: 0x3C and 0x3D)
   - ✅ Configurable content per display via `DISPLAY_LEFT_CONTENT` and `DISPLAY_RIGHT_CONTENT`
   - ✅ Available content types: `imu_raw_left`, `imu_raw_right`, `orientation_left`, `orientation_right`, `gps`
   - ✅ Default configuration: raw left IMU on left display, raw right IMU on right display
   - ✅ Configurable update interval (default: 250ms)
   - ✅ Splash screens on startup
   - ✅ Real-time MQTT subscription based on content configuration
   - ✅ 7x13 bitmap font rendering with direct pixel buffer manipulation
   - ✅ Entry point at `cmd/display/main.go`

7. **Add CLI options to ./cmd/xxx/main.go apps** ✅ PARTIALLY COMPLETED
   - ✅ Configuration system implemented via `inertial_config.txt`
   - ✅ All apps read config at startup (MQTT broker, ports, hardware settings)
   - ⚠️ Runtime CLI flags not yet implemented (could add flag overrides for config values)

---

## 9. MPU9250 Register Debugging Tool (Standalone Development)

### Overview
**STANDALONE APP** — Creates NEW files only, NO modifications to existing code.
MQTT-based low-level register interface for direct MPU9250 hardware tinkering.

### Status: ✅ COMPLETED (2025-01-03)

### Implementation Overview
Standalone WebSocket-based register debugging tool with comprehensive bitfield manipulation and live sensor monitoring.

### Files Created
- ✅ `cmd/register_debug/main.go` — standalone web server binary
- ✅ `internal/app/register_debug_handler.go` — WebSocket handler with register operations
- ✅ `internal/sensors/mpu9250_registers.go` — complete MPU9250 register map with bitfield definitions
- ✅ `web/register_debug.html` — interactive register debugging interface
- ✅ Register read/write functions integrated into `internal/sensors/imu.go` (IMUManager)

### Architecture
WebSocket-based direct hardware access, independent of MQTT architecture for low-latency debugging.

#### Web Server Component
Location: `cmd/register_debug/main.go`, `internal/app/register_debug_handler.go`

**WebSocket Communication** (port 8081 by default):
- Endpoint: `/ws` for bidirectional register operations
- Real-time sensor data via `/api/imu` REST endpoint
- Static web UI served at `/`

**Message Formats**:
- Read command: `{"action":"read","imu":"left","addr":"0x1B"}`
- Read all: `{"action":"read_all","imu":"left"}`
- Write command: `{"action":"write","imu":"left","addr":"0x1B","value":"0x10"}`
- Init command: `{"action":"init","imu":"left"}` — reinitialize IMU hardware
- Set SPI speed: `{"action":"set_spi_speed","imu":"left","read_speed":1000000,"write_speed":500000}`
- Export config: `{"action":"export_config","imu":"left"}` — export all registers as JSON
- Response: `{"type":"register_data","imu":"left","addr":"0x1B","value":"0x10","timestamp":"..."}`
- Bulk read: `{"type":"register_data","registers":{...all 128 registers...}}`
- Status: `{"type":"status","imu":"left","status":"initialized","read_speed":1000000,"write_speed":500000}`

**Hardware Access**:
- Uses existing IMUManager singleton via `sensors.GetIMUManager()`
- Shares hardware access with other components safely via mutex
- Runs as separate web server on different port from main web UI

#### Hardware Functions (IMUManager extensions)
Location: `internal/sensors/imu.go` (IMUManager methods)

**Implemented Functions**:
- ✅ `ReadRegister(imuID string, regAddr byte) (byte, error)` — single register read
- ✅ `WriteRegister(imuID string, regAddr byte, value byte) error` — single register write
- ✅ `ReadAllRegisters(imuID string) (map[byte]byte, error)` — bulk read 0x00-0x7F
- ✅ `GetRegisterMap() []RegisterInfo` — metadata with bitfield definitions
- ✅ `ReinitializeIMU(imuID string) error` — close and reopen SPI connection, reset IMU state
- ✅ `SetSPISpeed(imuID string, readSpeed, writeSpeed int64) error` — configure separate read/write speeds
- ✅ `GetSPISpeed(imuID string) (readSpeed, writeSpeed int64, error)` — query current SPI speeds
- ✅ `ExportRegisterConfig(imuID string) (RegisterConfigFile, error)` — export all registers as timestamped JSON

**Register Metadata** (`internal/sensors/mpu9250_registers.go`):
- Complete MPU9250 register map with 40+ registers defined
- Bitfield definitions for configuration registers (CONFIG, GYRO_CONFIG, ACCEL_CONFIG, etc.)
- Access control metadata (R/W/RW) for safety
- Default values and detailed descriptions

**Integration**: Uses existing IMUManager singleton, thread-safe via mutex.

#### Web UI Component  
Location: `web/register_debug.html`

**Implemented Features**:
- ✅ WebSocket client for real-time bidirectional communication
- ✅ IMU selector dropdown (left/right) with status indicators
- ✅ **"Read All Registers" button** — bulk read all 128 registers
- ✅ **Comprehensive register table** with:
  - Address (hex) | Name | Description | Current Value (hex/binary/decimal) | Actions
  - Expandable bitfield rows for detailed configuration
  - Individual bitfield toggle switches for writable registers
  - Write protection indicators (R/W/RW)
  - Highlighting for modified registers
- ✅ **Bitfield manipulation**:
  - Toggle switches for each bitfield in writable registers
  - Real-time bit value computation and preview
  - Automatic register value calculation from bitfield states
  - Apply button to write computed value
- ✅ **"Reinitialize IMU" button** — reset hardware without restarting application
- ✅ **SPI Speed controls**:
  - Input fields for read speed and write speed (Hz)
  - "Set SPI Speed" button with validation
  - Display current SPI speeds from status updates
  - Presets: Fast (4MHz/1MHz), Normal (1MHz/500kHz), Slow (500kHz/250kHz)
  - Speed preset quick-select buttons
- ✅ **Status card** showing:
  - IMU initialization state (left/right)
  - Current SPI speeds (read/write)
  - Connection status
- ✅ **Live sensor data cards** displaying real-time IMU readings:
  - Accelerometer (X, Y, Z) with current range setting
  - Gyroscope (X, Y, Z) with current range setting
  - Magnetometer (X, Y, Z) with field magnitude
  - Polling via REST API at 100ms intervals
- ✅ **Quick configuration presets**:
  - Factory defaults restoration
  - Common sensor configurations (high-precision, low-power, etc.)
  - Export current configuration as JSON file
- ✅ **Dark theme** matching main dashboard aesthetic

#### Endpoints
- **WebSocket**: `ws://localhost:8081/ws` — bidirectional register operations
- **REST API**: `http://localhost:8081/api/imu` — live sensor data (GET)
- **Static UI**: `http://localhost:8081/` — register debug interface
- **Navigation**: Link from main dashboard (`http://localhost:8080`) to register debugger

#### Configuration
No additional config required in `inertial_config.txt` — register debug runs on hardcoded port 8081.

**Future enhancement**: Could add `REGISTER_DEBUG_PORT` config option.
- `TOPIC_REGISTERS_STATUS` — IMU status updates
- `REGISTER_DEBUG_ALLOWED_RANGES` — safety: limit writable registers (e.g., "0x1B-0x1D,0x6B")
- `REGISTER_DEBUG_DEFAULT_READ_SPEED` — default SPI read speed in Hz (e.g., 1000000)
- `REGISTER_DEBUG_DEFAULT_WRITE_SPEED` — default SPI write speed in Hz (e.g., 500000)
- `REGISTER_DEBUG_MAX_SPI_SPEED` — maximum allowed SPI speed (e.g., 10000000)
- `REGISTER_DEBUG_MIN_SPI_SPEED` — minimum allowed SPI speed (e.g., 100000)

#### Safety Features
- ✅ **Read-only indicators**: Registers marked as "R" cannot be written
- ✅ **Bitfield value validation**: Ensures valid bit patterns before writing
- ✅ **Confirmation for critical registers**: PWR_MGMT, INT_PIN_CFG require confirmation
- ✅ **Current value display**: Shows register state before modification
- ✅ **Binary/hex/decimal views**: Multiple formats for easier debugging
- ✅ **Revert capability**: Can reload all registers to see current hardware state
- ✅ **Export/restore**: Save and load register configurations as JSON

### Use Cases
- ✅ **Experiment with sensor ranges**: Toggle ACCEL_FS_SEL and GYRO_FS_SEL bitfields to change ±2g/±4g/±8g/±16g and ±250°/s/±500°/s/±1000°/s/±2000°/s
- ✅ **Configure DLPF**: Adjust digital low-pass filter settings via CONFIG register bitfields
- ✅ **Test FIFO configuration**: Enable/disable FIFO and configure buffering
- ✅ **Debug I2C master settings**: Tune magnetometer communication via I2C_MST_CTRL bitfields
- ✅ **Validate interrupt pin configuration**: Modify INT_PIN_CFG bitfields for interrupt routing
- ✅ **Live observation**: Watch sensor data change in real-time as registers are modified
- ✅ **Recover from IMU lockup**: Use reinitialize button to reset without restarting application
- ✅ **Debug communication issues**: Try different SPI speeds to isolate timing problems
- ✅ **Optimize performance**: Use fast reads for monitoring, slower writes for reliability
- ✅ **Test register write timing**: Verify critical registers need slower SPI speeds
- ✅ **Export working configurations**: Save known-good register states for later restoration

### Benefits of WebSocket Architecture
- ✅ **Low latency**: Direct hardware access with minimal overhead
- ✅ **Real-time updates**: Instant feedback on register changes
- ✅ **Bidirectional**: Client can request and receive data in same connection
- ✅ **Stateful sessions**: Maintains connection state for efficient operations
- ✅ **Standalone operation**: Runs independently on separate port from main web UI
- ✅ **Zero MQTT dependency**: Simpler architecture for debugging tool
- ✅ **Browser-based**: No additional software installation required

### Integration
- ✅ **Navigation from main dashboard**: "Debug Registers" button in [index.html](web/index.html) header
- ✅ **Separate web server**: Runs on port 8081, independent of main web UI (port 8080)
- ✅ **Shared hardware access**: Uses IMUManager singleton safely with existing producers
- ✅ **No conflicts**: Can run simultaneously with imu_producer via mutex protection

### Running the Register Debugger
```bash
# Build the register debug tool
go build -o register_debug ./cmd/register_debug/

# Run with sudo (required for SPI hardware access)
sudo ./register_debug

# Access in browser
http://localhost:8081
```

**Note**: Can run concurrently with imu_producer - hardware access is thread-safe via IMUManager mutex.

---
