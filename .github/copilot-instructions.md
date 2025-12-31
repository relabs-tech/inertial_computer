# Inertial Computer – AI Agent Instructions

## Project Overview
Raspberry Pi-based inertial sensing platform using **MQTT message bus architecture**. Dual MPU9250 IMUs (left/right) with magnetometers, BMP sensors, and GPS publish sensor data over MQTT to multiple consumers (console, web UI, OLED displays). Go codebase targeting embedded Linux with periph.io hardware drivers.

## Architecture – Critical Patterns

### Message Bus Isolation
**Pattern**: Producers and consumers are completely decoupled via MQTT. Never add direct dependencies between them.
- **Producers** (`cmd/imu_producer`, `cmd/gps_producer`) read hardware and publish JSON
- **Consumers** (`cmd/console_mqtt`, `cmd/web`, `cmd/display`) subscribe to topics
- All components can restart independently without affecting others
- Example: Adding new consumer requires zero producer changes

### Configuration System
**CRITICAL**: `inertial_config.txt` is the single source of truth for ALL configurable values.
- **NEVER hardcode anything configurable**: MQTT brokers, topics, hardware paths, timing intervals, I2C addresses, sensor ranges, sample rates, display settings, port numbers, serial devices
- Load via `config.InitGlobal("./inertial_config.txt")` at startup (first line in main)
- Access via `config.Get()` singleton throughout application
- Example keys: `MQTT_BROKER`, `TOPIC_IMU_LEFT`, `IMU_LEFT_SPI_DEVICE`, `IMU_SAMPLE_INTERVAL`, `WEB_SERVER_PORT`, `GPS_SERIAL_PORT`, `DISPLAY_UPDATE_INTERVAL`
- **Adding new config**: Update `internal/config/config.go` struct, add to `inertial_config.txt`, use via `config.Get()`

### Hardware Abstraction Layer
**Critical boundary**: `internal/sensors/` is the ONLY package that imports periph.io.
- **Singleton pattern**: `IMUManager` maintains persistent hardware connections (call `GetIMUManager().Init()` once)
- Thread-safe with `sync.RWMutex` for concurrent sensor reads
- Graceful degradation: Individual IMU failures don't crash system; check `IsLeftIMUAvailable()`
- **Pure functions**: `orientation.AccelToPose()` is hardware-independent, takes floats, returns Pose

### Data Model Consistency
**Raw sensor data** (`internal/imu/imu_raw.go`):
```go
type IMURaw struct {
    Source string // "left" | "right"
    Ax, Ay, Az int16  // accelerometer (raw ADC)
    Gx, Gy, Gz int16  // gyroscope (raw ADC)
    Mx, My, Mz int16  // magnetometer (µT × 10)
}
```
Published to: `inertial/imu/left`, `inertial/imu/right`

**Orientation** (`internal/orientation/orientation.go`):
```go
type Pose struct {
    Roll, Pitch, Yaw float64 // degrees
}
```
Published to: `inertial/pose/left`, `inertial/pose/right`, `inertial/pose/fused`

## Development Workflows

### Building and Running
```bash
# Build individual components
go build -o imu_producer ./cmd/imu_producer/
go build -o web ./cmd/web/

# Run with sudo (required for GPIO/SPI/I2C access)
sudo ./imu_producer
./web  # web server doesn't need sudo

# Run with custom config
sudo ./imu_producer -config /path/to/custom_config.txt
```

### Testing Hardware Access
**Mock mode**: IMU producer supports mock data source for testing without hardware.
Toggle via `useMock` flag in `internal/app/imu_producer.go` (line ~50).

### MQTT Debugging
```bash
# Subscribe to all topics (requires mosquitto-clients)
mosquitto_sub -h localhost -t 'inertial/#' -v

# Test specific topic
mosquitto_sub -h localhost -t 'inertial/imu/left'

# Publish test message
mosquitto_pub -h localhost -t 'inertial/pose' -m '{"roll":12.5,"pitch":-5.3,"yaw":180.0}'
```

### Adding New Sensors
1. **Add configuration first**: Define all settings in `inertial_config.txt` (device paths, pins, timing, topics)
2. **Update config struct**: Add fields to `internal/config/config.go` and parsing logic
3. Add hardware interface to `internal/sensors/` (only place periph.io imports allowed)
4. Define data model in `internal/` subdirectory (e.g., `internal/temperature/`)
5. Create producer in `cmd/new_sensor/` calling `config.InitGlobal()` then reading config values
6. Publish JSON to MQTT topic (from config, never hardcoded)
7. Consumers auto-discover via MQTT subscription patterns

## Project-Specific Conventions

### Error Handling
- **Partial failures tolerated**: IMU manager allows one IMU to fail, system continues with available sensors
- **Log warnings, don't crash**: `fmt.Printf("Warning: Left IMU failed: %v\n", err)` then check availability
- **Hardware init failures**: Return errors but allow producers to continue with degraded capabilities

### Logging Format
Console output uses consistent prefixes for filtering:
```go
fmt.Printf("[IMU-L] ax=%5d ay=%5d az=%5d\n", raw.Ax, raw.Ay, raw.Az)
fmt.Printf("[GPS ] lat=%.6f lon=%.6f\n", fix.Latitude, fix.Longitude)
```

### Module Management
**Local fork**: `periph.io/x/devices/v3` replaced in `go.mod`:
```go
replace periph.io/x/devices/v3 => /home/dalarub/go/src/github.com/relabs-tech/devices
```
This fork includes custom MPU9250 magnetometer integration. Don't remove this replace directive.

### Deployment Scenarios
**Lab environment**: Controlled testing with stable hardware connections, full sensor suite
- All sensors available and calibrated
- MQTT broker, web UI, and displays running on same network
- Development and debugging with console output

**Field environment**: Portable deployment with potential sensor failures
- GPS required for outdoor navigation
- Network connectivity may be intermittent
- System must handle graceful degradation (missing sensors)
- Lower power consumption considerations

## Key Files Reference

- **Architecture deep dive**: [ARCHITECTURE.md](ARCHITECTURE.md) (983 lines) – complete system design, data flow, fusion strategy
- **Quick start**: [QUICKSTART.md](QUICKSTART.md) – building, MQTT setup, running components
- **Task tracking**: [TODO.md](TODO.md) – current status, known issues, next steps
- **Calibration**: [CALIBRATION_UI.md](CALIBRATION_UI.md) – web and CLI calibration tools
- **Config**: `inertial_config.txt` – all runtime settings
- **Hardware abstraction**: `internal/sensors/imu.go` (IMUManager), `internal/sensors/imu_source.go` (periph.io wrapper)
- **IMU producer**: `internal/app/imu_producer.go` – main sensor loop, MQTT publishing
- **Web server**: `internal/app/web.go` – REST API, WebSocket calibration
- **Display**: `internal/app/display.go` – dual SSD1306 OLED rendering

## Current Development State

**Working**: Dual IMU (accel/gyro/mag), GPS (full NMEA), BMP sensors, MQTT architecture, web UI with real-time dashboard, OLED displays, calibration UI framework

**Current Priority**: 
- **Calibration debugging/enhancement**: Fix and improve calibration procedures (gyro, accel, mag) before applying coefficients
- Both web-based (`/calibration.html`) and CLI (`cmd/calibration`) tools need validation
- Focus on calibration algorithm accuracy and user guidance improvements

**Expected Sensor Readings** (for calibration validation):
- **Accelerometer** (int16 raw ADC counts):
  - **Range depends on config**: IMU_ACCEL_RANGE in `inertial_config.txt` (0=±2g, 1=±4g, 2=±8g, 3=±16g)
  - **Default (±8g)**: Full scale = 65536 counts for 16g → ~4096 counts per g
  - **Static on flat surface**: Z-axis ≈ +4096 (1g up), X ≈ 0, Y ≈ 0 (±50 counts noise acceptable)
  - **Inverted (upside down)**: Z-axis ≈ -4096 (-1g down), X ≈ 0, Y ≈ 0
  - **Typical range**: -32768 to +32767 (int16), but normal operation rarely exceeds ±8192 for ±2g of actual acceleration
  - **6-point calibration**: Each axis should measure ≈ ±1g when pointing up/down (~±4096 for ±8g range)
- **Gyroscope** (int16 raw ADC counts, degrees per second):
  - **Range depends on config**: IMU_GYRO_RANGE (0=±250°/s, 1=±500°/s, 2=±1000°/s, 3=±2000°/s)
  - **Static bias**: Should be near zero but can drift ±50-200 counts; calibration removes this
  - **Dynamic rotation**: Visible large values during movement, returns to bias when still
- **Magnetometer** (int16, µT × 10):
  - **Earth's magnetic field**: ~25-65 µT depending on location (250-650 in stored units)
  - **Expected range**: -500 to +500 typical, varies by orientation and local magnetic environment
  - **Static reading**: Should show consistent magnitude (~300-500) but direction changes with device orientation
  - **Calibration note**: Hard-iron offset (bias) and soft-iron (scale) vary by environment; recalibrate when moved

**Display Units**:
- **Web UI** (`web/index.html`):
  - Orientation (Roll, Pitch, Yaw): degrees (`deg`)
  - Accelerometer (Ax, Ay, Az): raw int16 counts (no unit displayed)
  - Gyroscope (Gx, Gy, Gz): raw int16 counts (no unit displayed)
  - Magnetometer (Mx, My, Mz): raw int16 counts (µT × 10, no unit displayed)
  - Temperature: degrees Celsius (`°C`)
  - Pressure: hectopascals (`hPa`)
  - GPS altitude: meters (`m`)
  - GPS speed: knots (`kn`)
  - GPS course: degrees (`deg`)
- **OLED Display** (`internal/app/display.go`):
  - IMU raw mode: Shows raw int16 counts (no units), format `A:ax ay az G:gx gy gz`
  - Orientation mode: Shows degrees (no unit symbol), format `R: roll P: pitch Y: yaw`
  - GPS mode: Shows latitude/longitude (decimal degrees), altitude (meters)

**Next Steps** (after calibration):
1. Apply calibration coefficients to sensor readings in producers
2. Implement sensor fusion (gyro + mag → yaw)
3. Testing infrastructure (unit, integration, hardware simulation)

**Known Constraints**:
- Magnetometer accessed via MPU9250's internal I2C (EXT_SENS_DATA registers), not directly
- Producers require `sudo` for SPI/GPIO/I2C hardware access
- Web UI polls REST API at 500ms intervals (not WebSocket streaming except calibration)
- Yaw currently hardcoded to 0 (placeholder until mag fusion implemented)

## Common Pitfalls

1. **NEVER hardcode configurable values**: Every path, topic, port, interval, address MUST come from `inertial_config.txt`
   - ❌ Bad: `const port = 8080` or `topic := "inertial/imu/left"`
   - ✅ Good: `port := config.Get().WebServerPort` and `topic := config.Get().TopicIMULeft`
   - This is the #1 rule - violating it breaks multi-environment deployment
2. **Don't import periph.io outside `internal/sensors/`**: Breaks hardware abstraction and testability
3. **Don't couple producers to consumers**: All communication via MQTT only
4. **Remember `sudo`**: Hardware producers fail silently without GPIO/SPI permissions
5. **Don't modify data models without updating all consumers**: JSON schema changes require coordinated updates
