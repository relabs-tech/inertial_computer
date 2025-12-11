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

    Mx int16 `json:"mx"` // magnetometer X
    My int16 `json:"my"` // magnetometer Y
    Mz int16 `json:"mz"` // magnetometer Z
}
```

Published on:

- `inertial/imu/left`
- `inertial/imu/right`

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
- ✅ Left IMU gyroscope (rotation) reads implemented via `GetRotationX/Y/Z`
- ⚠️ Magnetometer (AK8963) reads from EXT_SENS_DATA still TODO
- ❌ Right IMU: not yet wired or implemented
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
- ✅ Left IMU accel and gyro reads implemented; magnetometer values still TODO
- ⚠️ BMP readings return zeros (driver TODO)

Additional runtime behavior:

- The inertial producer now logs actionable runtime output each tick: timestamped pose (roll/pitch/yaw) and the left IMU's raw accelerometer and gyroscope values are printed to stdout for ease of debugging and integration testing.

Future enhancements:

- Implement gyro driver calls → integrate angular velocity for yaw
- Add magnetometer driver calls → correct yaw with heading
- Implement complementary filter or EKF for robust fusion
- Add dual-IMU cross-validation

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

---

## 5. Consumers

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
- **Right IMU (MPU9250)**: TBD (not yet implemented)
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

- **IMU**: Left IMU reads accel/gyro from real MPU9250, but returns zeros for most reads (driver integration WIP)
- **Magnetometer**: Published but not fused; yaw is hardcoded to 0
- **Right IMU**: Not yet implemented; stub returns zeros
- **Environmental sensors**: Both left and right BMP stubs return zeros (driver integration needed)
- **Calibration**: No automatic or interactive calibration routines yet
- **Documentation**: Missing HARDWARE.md (pin assignments) and CALIBRATION.md (procedure)

See TODO.md for detailed prioritized task list.

