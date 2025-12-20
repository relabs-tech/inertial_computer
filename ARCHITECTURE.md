# Architecture – Inertial Computer

This document describes the internal architecture, data flow, and design rationale of the *Inertial Computer* project. It is intended for developers working on sensor integration, fusion algorithms, or system extensions.

---

## 1. Architectural overview

The system is built around **MQTT as a message bus**.

```text
┌──────────────┐
│ Sensors      │
│ IMU / BMP / │
│ GPS          │
└──────┬───────┘
       │
       ▼
┌────────────────┐
│ Producers      │
│ (cmd/*)        │
└──────┬─────────┘
       │ MQTT (JSON)
       ▼
┌────────────────┐
│ Mosquitto      │
│ Broker         │
└──────┬─────────┘
       │
       ├───────────────────┐
       ▼                   ▼
┌──────────────┐   ┌────────────────┐
│ Console      │   │ Web Server +    │
│ Subscriber   │   │ Web UI          │
└──────────────┘   └────────────────┘
```

Characteristics:

- producers and consumers are fully decoupled
- any component can be restarted independently
- multiple sinks can consume the same data

---

## 2. Domain model

### 2.1 Orientation

```go
type Pose struct {
    Roll  float64 `json:"roll"`
    Pitch float64 `json:"pitch"`
    Yaw   float64 `json:"yaw"`
}
```

Used for:

- raw orientation estimates
- fused orientation output

Published on:

- `inertial/pose`
- `inertial/pose/fused`

---

### 2.2 Raw IMU data

```go
type IMURaw struct {
    Source string `json:"source"` // "left" | "right"

    Ax int16 `json:"ax"` // accelerometer X
    Ay int16 `json:"ay"` // accelerometer Y
    Az int16 `json:"az"` // accelerometer Z

    Gx int16 `json:"gx"` // gyroscope X
    Gy int16 `json:"gy"` // gyroscope Y
    Gz int16 `json:"gz"` // gyroscope Z

    Mx int16 `json:"mx"` // magnetometer X (µT × 10)
    My int16 `json:"my"` // magnetometer Y (µT × 10)
    Mz int16 `json:"mz"` // magnetometer Z (µT × 10)
}
```

**Notes**:
- Magnetometer values are scaled as int16 (µT × 10) for consistency with other sensor readings
- All values are raw, uncalibrated sensor outputs
- Magnetometer reads may be zero if initialization failed (non-fatal)

Published on:

- `inertial/imu/left`
- `inertial/imu/right`

**Test/debug topic** (temporary):
- `inertial/mag/left` — magnetometer-only data with computed field magnitude

Format:
```json
{
  "mx": -180,
  "my": 210,
  "mz": -50,
  "norm": 285.3,
  "time": "2025-12-20T12:34:56Z"
}
```

This topic will be removed once magnetometer fusion is stable.

---

### 2.3 Environmental data (BMP)

```go
type Sample struct {
    Source      string  `json:"source"`       // "left" | "right"
    Temperature float64 `json:"temp_c"`       // temperature in °C
    Pressure    float64 `json:"pressure_pa"`  // atmospheric pressure in Pa
}
```

Published on:

- `inertial/bmp/left`
- `inertial/bmp/right`

---

### 2.4 GPS

```go
type Fix struct {
    Time       string  `json:"time"`
    Date       string  `json:"date"`
    Latitude   float64 `json:"lat"`
    Longitude  float64 `json:"lon"`
    SpeedKnots float64 `json:"speed_knots"`
    CourseDeg  float64 `json:"course_deg"`
    Validity   string  `json:"validity"`
}
```

Published on:

- `inertial/gps`

---

## 3. Hardware abstraction & sensor reading

All direct sensor access is isolated in `internal/sensors/` with a clean interface for pose computation:

```
internal/sensors/
  └─ imuSource
     └─ ReadRaw() → IMURaw {Ax, Ay, Az, Gx, Gy, Gz, Mx, My, Mz}
```

### Key interfaces and functions

**`IMURawReader` interface** (sensors package)
```go
type IMURawReader interface {
    ReadRaw() (imu_raw.IMURaw, error)
}
```

Implemented by `imuSource` which wraps an MPU9250 device. Returns raw accelerometer, gyroscope, and magnetometer data with no pose computation.

**Pose computation functions** (orientation package)
- `AccelToPose(ax, ay, az float64) Pose` — pure function
- `ComputePoseFromAccel(ax, ay, az float64) Pose` — same as above
- Both use simple tilt formulas; yaw hardcoded to 0

### Architecture benefits

- **Separation of concerns**: hardware reads separate from pose math
- **Testability**: pose functions are pure, no dependencies
- **Flexibility**: can compute poses from different sensors or sensor combinations
- **Extensibility**: ready for gyro integration, magnetometer fusion, multi-IMU blending

### Current status

- ✅ Left IMU reads accel via SPI (MPU9250)
- ✅ Left IMU gyro reads implemented via `GetRotationX/Y/Z()`
- ✅ Left IMU magnetometer (AK8963) initialized and reading via internal I2C
- ✅ Right IMU reads accel via SPI (MPU9250, wired and tested)
- ✅ Right IMU gyro reads implemented via `GetRotationX/Y/Z()`
- ✅ Right IMU magnetometer (AK8963) initialized and reading via internal I2C
- ⚠️ Magnetometer calibration not yet implemented (hard-iron/soft-iron correction TODO)
- ⚠️ Magnetometer data not yet used in yaw calculation (fusion TODO)
- ❌ BMP sensors: stubs return zero values

The rest of the system **never** imports periph.io or hardware-specific code directly.

---

## 4. Producers

### 4.1 Inertial producer (`cmd/producer`)

Entry point: `internal/app/RunInertialProducer()`

**Refactored architecture** (branch: `refactor/separate-raw-reads-from-pose`):

Responsibilities:

- choose data source (mock or real IMU)
- connect to MQTT broker
- loop every 100ms:
  - **mock path**: call `mockSrc.Next()` → get pose directly
  - **real IMU path**: 
    1. call `imuReader.ReadRaw()` → get raw IMU data (int16 values)
    2. convert to float64 and call `orientation.AccelToPose()` → get pose
  - publish pose to `inertial/pose` and `inertial/pose/fused`
  - read/publish left/right raw IMU via `sensors.ReadLeftIMURaw()`, `sensors.ReadRightIMURaw()`
  - read/publish left/right BMP via `sensors.ReadLeftEnv()`, `sensors.ReadRightEnv()`

Current implementation:

- ✅ Uses `IMURawReader` interface for hardware decoupling
- ✅ Pure pose computation via `orientation.AccelToPose()`
- ✅ Mock mode still works (toggle via `useMock` flag)
- ✅ Left and right IMU accel, gyro, and mag readings published each tick
- ✅ Producer logs pose, accel, gyro, and mag (with magnitude) each 100ms tick
- ✅ Test/debug MQTT topic `inertial/mag/left` publishes magnetometer data with field magnitude
- ✅ Magnetometer reads scaled as int16 (µT * 10) for consistency
- ⚠️ Magnetometer calibration not applied (hard-iron/soft-iron correction TODO)
- ⚠️ Magnetometer data not yet integrated into yaw calculation
- ⚠️ BMP readings return zeros (driver TODO)

Future enhancements:

- Implement magnetometer calibration (hard-iron and soft-iron correction)
- Integrate gyro angular velocity for dynamic yaw estimation
- Add magnetometer fusion to correct yaw drift with heading
- Implement complementary filter or EKF for robust sensor fusion
- Add dual-IMU cross-validation and fusion

---

### 4.2 GPS producer (`cmd/gps_producer`)

Entry point: `internal/app/RunGPSProducer()`

Responsibilities:

- open GPS serial port (defaults to `/dev/serial0` at 9600 baud)
- read and parse NMEA sentences from GPS module
- extract RMC (Recommended Minimum Sentence) messages
- populate GPS Fix struct with:
  - time, date, latitude, longitude
  - speed over ground (knots)
  - course over ground (degrees)
  - validity indicator
- publish complete fix as JSON to `inertial/gps`

Current implementation:

- uses `github.com/adrianmo/go-nmea` for parsing
- uses `github.com/jacobsa/go-serial` for serial I/O
- reads sentences in a loop and publishes each RMC as a separate fix
- skips unparseable sentences silently

Future enhancements:

- support additional NMEA sentences (GGA for altitude, GSA for fix quality)
- implement time synchronization
- add data validation and outlier detection

## 5. Consumers

### 5.1 Console MQTT subscriber (`cmd/console_mqtt`)

Entry point: `internal/app/RunConsoleMQTT()`

Responsibilities:

- connect to MQTT broker
- subscribe to all data streams
- decode JSON payloads into domain structs
- print formatted human-readable output to stdout

Output format:

```
[POSE]  ROLL=  20.45  PITCH=  -5.12  YAW= 123.67
[FUSE] ROLL=  20.45  PITCH=  -5.12  YAW= 123.67
[IMU-L] ax=    145 ay=   -230 az=  9850  gx=    10 gy=   -15 gz=    -5  mx=   -180 my=   210 mz=   -50
[IMU-R] ax=    150 ay=   -225 az=  9855  gx=    12 gy=   -18 gz=    -3  mx=   -175 my=   215 mz=   -48
[GPS ]  time=12:34:56 date=2025-12-09 lat=40.712776 lon=-74.005974 speed=5.2kn course=45.3° validity=A
```

Used primarily for:

- real-time monitoring and debugging
- validation during sensor integration
- headless operation without web UI

---

### 5.2 Web server (`cmd/web`)

Entry point: `internal/app/RunWeb()`

Responsibilities:

- connect to MQTT broker
- subscribe to all data streams
- maintain in-memory cache of latest values per stream (protected by RWMutex)
- expose REST-style JSON APIs:

```
GET /api/orientation          → last Pose
GET /api/orientation/fused    → last fused Pose
GET /api/imu/left             → last left IMURaw
GET /api/imu/right            → last right IMURaw
GET /api/env/left             → last left Sample (temp + pressure)
GET /api/env/right            → last right Sample (temp + pressure)
GET /api/gps                  → last GPS Fix
```

- serve static HTML/JS dashboard from `web/` directory on port 8080

Frontend behavior:

- polls all APIs every 500ms
- updates live dashboard with latest values
- displays connection status for each stream
- renders 8 cards in a responsive grid layout
- uses dark theme with accent lighting

Future enhancements:

- WebSocket support for lower-latency updates
- 3D visualization (three.js) for orientation
- time-series graphs and data logging
- stream health indicators

---

## 6. Fusion strategy (current and future)

### Current state

- **raw pose** (`inertial/pose`): derived from left IMU accelerometer only
  - roll and pitch computed via simple tilt formulas: `atan2(ay, az)` and `atan2(-ax, sqrt(ay² + az²))`
  - yaw hardcoded to 0 (placeholder)
  
- **fused pose** (`inertial/pose/fused`): currently identical to raw pose
  - will become true sensor fusion output once algorithms are implemented

- **raw IMU data** published but magnetometer values not yet used for yaw stabilization

### Planned evolution

1. **Phase 1: Basic gyro integration**
   - integrate gyroscope angular velocity over time
   - use accelerometer for drift correction
   - establish baseline yaw from magnetometer when device is stationary

2. **Phase 2: Magnetometer yaw alignment**
   - read magnetometer from EXT_SENS_DATA registers
   - compute heading via atan2(My, Mx) after soft-iron calibration
   - blend gyro-integrated yaw with magnetometer heading

3. **Phase 3: Dual-IMU fusion**
   - cross-validate left and right IMU readings
   - detect sensor drift or faults
   - combine readings for improved accuracy

4. **Phase 4: Advanced fusion (optional)**
   - implement complementary filter or extended Kalman filter (EKF)
   - incorporate GPS course for outdoor navigation
   - add wheel odometry or other sensors if available

### Design constraint

Fusion algorithms remain **internal to the producer**. Consumers always receive finished pose data and do not need to understand fusion details.

---

## 7. Deployment topology

### Single-machine setup (development/testing)

```
┌─────────────────────────────────┐
│ Raspberry Pi                    │
├─────────────────────────────────┤
│ [producer]  [gps_producer]      │ ← cmd/* entry points
│      │             │            │
│      └─────────┬───┘            │
│              MQTT               │
│      ┌────────┴────────┐        │
│      ▼                 ▼        │
│  [mosquitto]    [web] [console]│
│  (broker)       (REST)  (MQTT) │
│                  │             │
│              port 8080         │
└──────────────────┬─────────────┘
                   │
         ┌─────────┴─────────┐
         ▼                   ▼
    [browser]          [headless monitor]
    (web UI)           (console output)
```

### Key assumptions

- All producers and consumers run on the same Pi
- Mosquitto broker runs locally on port 1883
- Web UI accessible via browser on same network
- Serial GPS on `/dev/serial0` (or configurable)

### Hardware connections

- **Left IMU (MPU9250)**: SPI2 (`/dev/spidev6.0`), CS on GPIO 18
- **Right IMU (MPU9250)**: SPI0 (`/dev/spidev0.0`), CS on GPIO 8
- **Environmental sensors (BMP)**: I2C (address TBD)
- **GPS module**: Serial port, 9600 baud

---

## 8. Key architectural decisions

- **MQTT** chosen over direct RPC to simplify fan-out and allow independent restart of components
- **JSON** used for transport to maximize debuggability and enable inspection/logging
- **Strict layering** enforces separation between domain models, transport, and hardware:
  - Domain structs (`internal/{orientation,imu,env,gps}`) have no dependencies
  - Sensor layer (`internal/sensors`) isolated from app logic
  - Producers/consumers (`internal/app`) never directly access hardware
- **`internal/` package** used to prevent external consumers from coupling to implementation details
- **Pull-based REST API** for web UI instead of WebSockets (simpler, works behind proxies)

---

## 9. Extension ideas

- **Data logging + replay mode** for offline analysis and algorithm testing
- **WebSocket streaming** for real-time low-latency updates
- **3D visualization** (three.js) for intuitive orientation display
- **Time-series graphs** for IMU/BMP trends
- **Stream health monitoring** (staleness detection, error counters)
- **Additional sensors**: airspeed, wheel encoders, barometer, magnetometer standalone
- **Multi-producer coordination**: synchronized reads from multiple IMUs
- **Calibration tools**: magnetometer soft-iron/hard-iron compensation, IMU factory calibration
- **Cloud export**: optional MQTT bridge to cloud services for long-term analysis

---

## 10. Guiding principles

> Add sensors freely.
> Fuse intelligently.
> Consume everywhere.
> Never couple producers to consumers.

---

## 11. Known limitations & TODOs

- **Left IMU**: Reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- **Right IMU**: Reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- **Magnetometer calibration**: Hard-iron and soft-iron correction not yet implemented
- **Yaw calculation**: Currently hardcoded to 0; needs gyro integration and magnetometer fusion
- **Environmental sensors**: Both left and right BMP stubs return zeros (driver integration needed)

---

## 12. Recent changes (Mag_Add branch)

### Magnetometer integration (commits: 139a91d, f5f3cb3)

**Driver changes** (`internal/sensors/imu_source.go`):
- Added `InitMag()` call during IMU initialization for both left and right sensors
- Magnetometer initialization is non-fatal; system continues if mag unavailable
- Added `magCal` field to store magnetometer calibration parameters
- Added `magReady` flag to track whether magnetometer is operational
- `ReadRaw()` now calls `imu.ReadMag(magCal)` to retrieve magnetometer data
- Raw magnetometer values scaled as int16 (µT × 10) for consistency with accel/gyro
- Overflow detection implemented; overflows are silently skipped

**Producer test/debug code** (`internal/app/imu_producer.go`):
- Added `magNorm()` helper function to compute magnetic field magnitude
- New MQTT topic `inertial/mag/left` publishes magnetometer-only data with:
  - Raw mx, my, mz values
  - Computed field magnitude (|B|)
  - RFC3339 timestamp
- Updated logging to include magnetometer readings: `mx=X my=Y mz=Z |B|=N`
- Test code allows validation of magnetometer behavior before fusion integration

**Dependencies** (`go.mod`):
- Added local replace directive: `periph.io/x/devices/v3` → local fork with magnetometer support
- This temporary change supports testing of magnetometer driver enhancements

**Next steps**:
1. Implement hard-iron and soft-iron calibration routines
2. Apply calibration to raw magnetometer readings
3. Integrate calibrated magnetometer data into yaw calculation
4. Remove test/debug MQTT topic once fusion is stable
5. Update `go.mod` to use upstream periph.io once magnetometer support is merged
- **Calibration**: No automatic or interactive calibration routines yet
- **Documentation**: Missing HARDWARE.md (pin assignments) and CALIBRATION.md (procedure)

See TODO.md for detailed prioritized task list.

