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

### Status: ⚠️ PLANNED

### New Files to Create
- **NEW**: `cmd/register_debug/main.go` — standalone producer binary
- **NEW**: `internal/app/register_debug.go` — register debug app logic
- **NEW**: `internal/sensors/imu_registers.go` — low-level register read/write functions
- **NEW**: `web/register_debug.html` — register debugging web interface
- **NO modifications** to existing producers, consumers, or sensor files

### Architecture
Follows standard MQTT producer/consumer isolation pattern with complete independence from existing codebase.

#### Producer Component
Location: `cmd/register_debug/main.go`, `internal/app/register_debug.go`

**MQTT Communication**:
- Subscribes to command topics: `inertial/registers/cmd/read`, `inertial/registers/cmd/write`, `inertial/registers/cmd/init`, `inertial/registers/cmd/spi_speed`
- Publishes responses to: `inertial/registers/data/left`, `inertial/registers/data/right`
- Publishes register map metadata to: `inertial/registers/map`
- Publishes status updates to: `inertial/registers/status`

**Message Formats**:
- Read/Write command: `{"imu":"left","addr":"0x1B","value":"0x10"}` (value only for writes)
- Init command: `{"imu":"left"}` — reinitialize IMU hardware
- SPI speed command: `{"imu":"left","read_speed":"1000000","write_speed":"500000"}` — set speeds in Hz
- Response: `{"imu":"left","addr":"0x1B","value":"0x10","timestamp":"..."}`
- Bulk read: `{"imu":"left","registers":{...all 128 registers...}}`
- Status response: `{"imu":"left","status":"initialized","read_speed":1000000,"write_speed":500000}`

**Hardware Access**:
- Creates its own periph.io SPI connections (does NOT modify existing IMUManager)
- Can reference `config.Get()` for device paths, but uses independent hardware access
- Runs as separate process from imu_producer

#### Low-Level Hardware Functions
Location: NEW file `internal/sensors/imu_registers.go`

Functions to implement:
- `ReadRegister(imuID string, regAddr byte) (byte, error)` — single register read
- `WriteRegister(imuID string, regAddr byte, value byte) error` — single register write
- `ReadAllRegisters(imuID string) (map[byte]byte, error)` — bulk read 0x00-0x7F
- `GetRegisterMap() []RegisterInfo` — metadata (name, address, description, R/W/RW)
- `ReinitializeIMU(imuID string) error` — close and reopen SPI connection, reset IMU state
- `SetSPISpeed(imuID string, readSpeed, writeSpeed int64) error` — configure separate speeds for read/write operations
- `GetSPISpeed(imuID string) (readSpeed, writeSpeed int64, err error)` — query current SPI speeds

**Key constraint**: Creates its own periph.io SPI connections, completely separate from existing sensor layer.

#### Web UI Component
Location: NEW file `web/register_debug.html`

**Features**:
- Pure MQTT consumer (subscribes via JavaScript MQTT client)
- IMU selector dropdown (left/right)
- "Read All Registers" button — publishes command to `inertial/registers/cmd/read`
- Register table: Address (hex) | Name | Description | Current Value | Write Value | R/W
- Input fields for writing new values to registers
- "Write Register" button — publishes command to `inertial/registers/cmd/write`
- **"Reinitialize IMU" button** — publishes command to `inertial/registers/cmd/init` to reset hardware
- **SPI Speed controls**:
  - Input fields for read speed and write speed (Hz)
  - "Set SPI Speed" button — publishes command to `inertial/registers/cmd/spi_speed`
  - Display current SPI speeds from status updates
  - Presets: Fast (4MHz/1MHz), Normal (1MHz/500kHz), Slow (500kHz/250kHz)
- Status indicator showing IMU initialization state and current SPI speeds
- Card widgets for live sensor data (accel, gyro, mag) subscribing to existing IMU topics
- Subscribes to `inertial/registers/data/#` for register responses
- Subscribes to `inertial/registers/map` for metadata (loaded once on page load)
- Subscribes to `inertial/registers/status` for initialization and speed updates

#### MQTT Topics
- `inertial/registers/cmd/read` — command: read register(s)
- `inertial/registers/cmd/write` — command: write register
- `inertial/registers/cmd/init` — command: reinitialize IMU hardware
- `inertial/registers/cmd/spi_speed` — command: set SPI read/write speeds
- `inertial/registers/data/left` — left IMU register data responses
- `inertial/registers/data/right` — right IMU register data responses
- `inertial/registers/map` — register metadata (published on producer startup)
- `inertial/registers/status` — IMU status (initialization state, SPI speeds)
- Reuses existing topics for live sensor data: `inertial/imu/left`, `inertial/imu/right`

#### Configuration
Add to `inertial_config.txt`:
- `TOPIC_REGISTERS_CMD_READ` — command topic for reads
- `TOPIC_REGISTERS_CMD_WRITE` — command topic for writes
- `TOPIC_REGISTERS_CMD_INIT` — command topic for IMU reinitialization
- `TOPIC_REGISTERS_CMD_SPI_SPEED` — command topic for SPI speed control
- `TOPIC_REGISTERS_DATA_LEFT` — left IMU register data
- `TOPIC_REGISTERS_DATA_RIGHT` — right IMU register data
- `TOPIC_REGISTERS_MAP` — register metadata
- `TOPIC_REGISTERS_STATUS` — IMU status updates
- `REGISTER_DEBUG_ALLOWED_RANGES` — safety: limit writable registers (e.g., "0x1B-0x1D,0x6B")
- `REGISTER_DEBUG_DEFAULT_READ_SPEED` — default SPI read speed in Hz (e.g., 1000000)
- `REGISTER_DEBUG_DEFAULT_WRITE_SPEED` — default SPI write speed in Hz (e.g., 500000)
- `REGISTER_DEBUG_MAX_SPI_SPEED` — maximum allowed SPI speed (e.g., 10000000)
- `REGISTER_DEBUG_MIN_SPI_SPEED` — minimum allowed SPI speed (e.g., 100000)

#### Safety Features
- Read-only mode toggle (prevent accidental writes)
- Producer validates writes against allowed ranges before executing
- Confirmation modal for writes to critical registers (PWR_MGMT, INT_PIN_CFG)
- Register value validation (range checks, reserved bit warnings)
- "Reset to Default" button publishes write command with factory defaults

### Use Cases
- Experiment with sensor ranges (accel ±2g/±4g/±8g/±16g)
- Test FIFO configuration and data rates
- Debug I2C master settings for magnetometer communication
- Validate interrupt pin configuration
- Low-pass filter parameter tuning
- Live observation of register effects on sensor readings
- **Recover from IMU lockup or misconfiguration** — reinitialize hardware without restarting producer
- **Debug communication issues** — try different SPI speeds to isolate timing problems
- **Optimize performance** — use fast reads for monitoring, slower writes for reliability
- **Test register write timing** — verify critical registers need slower SPI speeds

### Benefits of MQTT Architecture
- Producer and web UI completely decoupled
- Can restart web server without disturbing hardware access
- Multiple clients can monitor register changes simultaneously
- Can add CLI consumer for scripted register manipulation
- Command history can be logged by separate MQTT subscriber
- Zero impact on existing producers/consumers

### Integration
- Link from main page: Add "Debug Registers" navigation link in [index.html](web/index.html)
- No changes required to existing applications

---
