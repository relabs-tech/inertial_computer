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
       ├───────────────────┬──────────────┐
       ▼                   ▼              ▼
┌──────────────┐   ┌────────────────┐  ┌──────────────┐
│ Console      │   │ Web Server +    │  │ Display      │
│ Subscriber   │   │ Web UI          │  │ Consumer     │
└──────────────┘   └────────────────┘  │ (OLED x2)    │
                                        └──────────────┘
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

---

### 2.3 GPS Data

**Position & Velocity** (from RMC/GGA/VTG sentences):
```go
type Fix struct {
    Latitude   float64
    Longitude  float64
    Altitude   float64 // meters
    Speed      float64 // knots
    Course     float64 // degrees
    // ... quality metrics ...
}
```

**Satellite Visibility** (from GPGSV/GLGSV sentences):
```go
type SatellitesInView struct {
    GPSSatellites     []Satellite
    GLONASSSatellites []Satellite
    GPSCount          int
    GLONASSCount      int
}

type Satellite struct {
    PRN       int    // Satellite ID
    Elevation int    // degrees (0-90)
    Azimuth   int    // degrees (0-359)
    SNR       int    // signal-to-noise ratio (dBHz)
}
```

**Published Topics**:
- `inertial/gps` — position and velocity
- `inertial/gps/satellites` — GPS constellation only (circles in UI)
- `inertial/glonass/satellites` — GLONASS constellation only (squares in UI)

**Key Design**: GPS and GLONASS satellites are processed and published separately to prevent data contamination. Each topic receives only constellation-specific data using anonymous structs, ensuring clean payloads without cross-constellation fields.

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

Enhanced GPS data model with support for comprehensive NMEA sentence parsing:

```go
// Full GPS fix (legacy/combined topic)
type Fix struct {
    // From RMC (Recommended Minimum)
    Time       string  `json:"time"`
    Date       string  `json:"date"`
    Latitude   float64 `json:"lat"`
    Longitude  float64 `json:"lon"`
    SpeedKnots float64 `json:"speed_knots"`
    CourseDeg  float64 `json:"course_deg"`
    Validity   string  `json:"validity"`

    // From GGA (Global Positioning System Fix Data)
    Altitude      float64 `json:"altitude_m"`
    FixQuality    string  `json:"fix_quality"`
    NumSatellites int64   `json:"num_satellites"`
    HDOP          float64 `json:"hdop"`

    // From GSA (GPS DOP and Active Satellites)
    FixType string  `json:"fix_type"`
    PDOP    float64 `json:"pdop"`
    VDOP    float64 `json:"vdop"`

    // From VTG (Track Made Good and Ground Speed)
    SpeedKmh float64 `json:"speed_kmh"`

    // From GSV (GPS Satellites in View)
    SatellitesInView []Satellite `json:"satellites_in_view"`
}

// Position data (separate topic)
type Position struct {
    Time      string  `json:"time"`
    Date      string  `json:"date"`
    Latitude  float64 `json:"lat"`
    Longitude float64 `json:"lon"`
    Altitude  float64 `json:"altitude_m"`
    Validity  string  `json:"validity"`
}

// Velocity data (separate topic)
type Velocity struct {
    SpeedKnots float64 `json:"speed_knots"`
    SpeedKmh   float64 `json:"speed_kmh"`
    CourseDeg  float64 `json:"course_deg"`
}

// Quality metrics (separate topic)
type Quality struct {
    FixType       string  `json:"fix_type"`
    FixQuality    string  `json:"fix_quality"`
    NumSatellites int64   `json:"num_satellites"`
    HDOP          float64 `json:"hdop"`
    PDOP          float64 `json:"pdop"`
    VDOP          float64 `json:"vdop"`
}

// Satellite tracking (separate topic)
type Satellite struct {
    SVNumber  int64 `json:"sv_number"`  // satellite vehicle number (PRN)
    Elevation int64 `json:"elevation"`  // elevation in degrees (0-90)
    Azimuth   int64 `json:"azimuth"`    // azimuth in degrees (0-359)
    SNR       int64 `json:"snr"`        // signal-to-noise ratio in dB (0-99)
}

type SatellitesInView struct {
    Satellites []Satellite `json:"satellites"`
    Count      int         `json:"count"`
}
```

Published on:

- `inertial/gps` — full GPS data (all fields, legacy compatibility)
- `inertial/gps/position` — position and time only
- `inertial/gps/velocity` — speed and course only
- `inertial/gps/quality` — fix quality and DOP metrics
- `inertial/gps/satellites` — satellite visibility with signal strength

---

## 3. Configuration

All system settings are externalized to **`inertial_config.txt`** at the project root. This centralized configuration file eliminates hardcoded values and enables flexible deployment.

### Configuration Structure

The config file uses a simple `KEY=VALUE` format with `#` for comments:

```
# MQTT Settings
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID_PRODUCER=inertial_producer
MQTT_CLIENT_ID_GPS=gps_producer
MQTT_CLIENT_ID_CONSOLE=console_subscriber
MQTT_CLIENT_ID_WEB=web_subscriber

# MQTT Topics
TOPIC_POSE=inertial/pose
TOPIC_POSE_FUSED=inertial/pose/fused
TOPIC_IMU_LEFT=inertial/imu/left
TOPIC_IMU_RIGHT=inertial/imu/right
TOPIC_MAG_LEFT=inertial/mag/left
TOPIC_MAG_RIGHT=inertial/mag/right
TOPIC_BMP_LEFT=inertial/bmp/left
TOPIC_BMP_RIGHT=inertial/bmp/right
TOPIC_GPS_POSITION=inertial/gps/position
TOPIC_GPS_VELOCITY=inertial/gps/velocity
TOPIC_GPS_QUALITY=inertial/gps/quality
TOPIC_GPS_SATELLITES=inertial/gps/satellites
TOPIC_GPS=inertial/gps

# IMU Hardware
IMU_LEFT_SPI_DEVICE=/dev/spidev6.0
IMU_LEFT_CS_PIN=18
IMU_RIGHT_SPI_DEVICE=/dev/spidev0.0
IMU_RIGHT_CS_PIN=8
IMU_ACCEL_RANGE=2
IMU_GYRO_RANGE=1

# BMP Hardware
BMP_LEFT_SPI_DEVICE=/dev/spidev6.1
BMP_LEFT_PRESSURE_OSR=5
BMP_LEFT_TEMP_OSR=2
BMP_LEFT_IIR_FILTER=3
BMP_LEFT_STANDBY_TIME=1
BMP_RIGHT_SPI_DEVICE=/dev/spidev0.1
BMP_RIGHT_PRESSURE_OSR=5
BMP_RIGHT_TEMP_OSR=2
BMP_RIGHT_IIR_FILTER=3
BMP_RIGHT_STANDBY_TIME=1

# GPS Hardware
GPS_SERIAL_PORT=/dev/serial0
GPS_BAUD_RATE=9600

# Timing
IMU_SAMPLE_INTERVAL=100
CONSOLE_LOG_INTERVAL=1000

# Web Server
WEB_SERVER_PORT=8080
WEATHER_UPDATE_INTERVAL_MINUTES=5
```

### Implementation

- **Package**: `internal/config/config.go`
- **Initialization**: All apps call `config.InitGlobal("inertial_config.txt")` at startup
- **Access**: Components use `config.Get()` to retrieve the global singleton
- **Validation**: Required fields are checked at load time; missing values cause startup failure
- **Type Support**: String, int, bool with automatic conversion

This architecture ensures:
- No hardcoded hardware paths or MQTT topics in code
- Easy reconfiguration for different deployments
- Single source of truth for all system settings
- Compile-time independence from deployment details

---

## 4. Hardware abstraction & sensor reading

All direct sensor access is isolated in `internal/sensors/` with a clean interface for pose computation:

```
internal/sensors/
  └─ IMUManager (singleton)
     ├─ Init() — initialize both left and right IMU hardware
     ├─ ReadLeftIMU() → IMURaw {Ax, Ay, Az, Gx, Gy, Gz, Mx, My, Mz}
     ├─ ReadRightIMU() → IMURaw {Ax, Ay, Az, Gx, Gy, Gz, Mx, My, Mz}
     ├─ IsLeftIMUAvailable() → bool
     └─ IsRightIMUAvailable() → bool
```

### Key interfaces and functions

**`IMURawReader` interface** (sensors package)
```go
type IMURawReader interface {
    ReadRaw() (imu_raw.IMURaw, error)
}
```

Implemented by `imuSource` which wraps an MPU9250 device. Returns raw accelerometer, gyroscope, and magnetometer data with no pose computation.

**IMU Manager Pattern**

The system uses a singleton `IMUManager` to manage persistent hardware access:
- **Initialization**: `GetIMUManager()` returns singleton, `Init()` called once at startup
- **Thread Safety**: `sync.RWMutex` protects concurrent access
- **Hardware Persistence**: Sensors initialized once, reused across all reads
- **Graceful Degradation**: Individual IMU failures don't crash the system; availability checked via `IsLeftIMUAvailable()` / `IsRightIMUAvailable()`
- **Configuration-Driven**: SPI devices and CS pins read from `inertial_config.txt`

**Pose computation functions** (orientation package)
- `AccelToPose(ax, ay, az float64) Pose` — pure function
- `ComputePoseFromAccel(ax, ay, az float64) Pose` — same as above
- Both use simple tilt formulas; yaw hardcoded to 0

### Architecture benefits

- **Separation of concerns**: hardware reads separate from pose math
- **Testability**: pose functions are pure, no dependencies
- **Flexibility**: can compute poses from different sensors or sensor combinations
- **Extensibility**: ready for gyro integration, magnetometer fusion, multi-IMU blending
- **Configuration-driven**: all hardware settings externalized to config file
- **Singleton pattern**: efficient hardware access without re-initialization
- **Thread safety**: concurrent reads protected by mutex

### Current status

- ✅ Left IMU reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- ✅ Right IMU reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- ✅ IMU manager singleton pattern implemented for persistent hardware access
- ✅ Configuration system with `inertial_config.txt` for all hardware and MQTT settings
- ✅ BMP sensors (BMP280/BMP388) reading temperature and pressure via SPI with bmxx80 driver
- ✅ Pressure output in multiple units: Pa, mbar, and hPa
- ⚠️ Magnetometer calibration not yet implemented (hard-iron/soft-iron correction TODO)
- ⚠️ Magnetometer data not yet used in yaw calculation (fusion TODO)

The rest of the system **never** imports periph.io or hardware-specific code directly.

---

## 5. Producers

### 5.1 Inertial producer (`cmd/imu_producer`)

Entry point: `internal/app/RunInertialProducer()`

**Architecture**:

Responsibilities:

- read configuration from `inertial_config.txt`
- initialize IMU manager singleton
- choose data source (mock or real IMU)
- connect to MQTT broker
- loop every `IMU_SAMPLE_INTERVAL` (configurable, default 100ms):
  - **mock path**: call `mockSrc.Next()` → get pose directly
  - **real IMU path**: 
    1. call `imuManager.ReadLeftIMU()` and `imuManager.ReadRightIMU()` → get raw IMU data (int16 values)
    2. convert to float64 and call `orientation.AccelToPose()` → get pose
  - publish pose to configured topics (default: `inertial/pose` and `inertial/pose/fused`)
  - publish left/right raw IMU data with accel, gyro, and mag
  - publish left/right magnetometer-only data to dedicated topics
  - read/publish left/right BMP temperature and pressure (Pa, mbar, hPa)
- log consolidated sensor data at configurable interval (`CONSOLE_LOG_INTERVAL`)

Current implementation:

- ✅ Uses `IMUManager` singleton for persistent hardware access
- ✅ Pure pose computation via `orientation.AccelToPose()`
- ✅ Mock mode still works (toggle via `useMock` flag)
- ✅ Left and right IMU accel, gyro, and mag readings published each tick
- ✅ Dedicated magnetometer topics `inertial/mag/left` and `inertial/mag/right`
- ✅ BMP sensors fully integrated with bmxx80 driver
- ✅ Temperature and pressure published in multiple units
- ✅ Configuration-driven: MQTT topics, hardware paths, timing intervals from config file
- ✅ Configurable console logging (e.g., once per second instead of every tick)
- ⚠️ Magnetometer calibration not applied (hard-iron/soft-iron correction TODO)
- ⚠️ Magnetometer data not yet integrated into yaw calculation
- ⚠️ BMP readings functional but calibration coefficients not user-adjustable

Future enhancements:

- Implement magnetometer calibration (hard-iron and soft-iron correction)
- Integrate gyro angular velocity for dynamic yaw estimation
- Add magnetometer fusion to correct yaw drift with heading
- Implement complementary filter or EKF for robust sensor fusion
- Add dual-IMU cross-validation and fusion

---

### 5.2 GPS producer (`cmd/gps_producer`)

Entry point: `internal/app/RunGPSProducer()`

Responsibilities:

- read GPS serial port and baud rate from `inertial_config.txt`
- open GPS serial port (configurable, default `/dev/serial0` at 9600 baud)
- read and parse NMEA sentences from GPS module (RMC, GGA, GSA, VTG, GSV)
- extract comprehensive GPS data including:
  - position and time (from RMC and GGA)
  - speed and course (from RMC and VTG)
  - altitude above sea level (from GGA)
  - fix quality and DOP metrics (from GGA and GSA)
  - satellite visibility with signal strength (from GSV)
- publish to multiple MQTT topics for topic-based filtering:
  - `inertial/gps/position` — position, time, and altitude
  - `inertial/gps/velocity` — speed and course
  - `inertial/gps/quality` — fix type, quality, and DOP values
  - `inertial/gps/satellites` — satellites in view with elevation, azimuth, SNR
  - `inertial/gps` — full combined data (legacy compatibility)

Current implementation:

- ✅ uses `github.com/adrianmo/go-nmea` for parsing
- ✅ uses `github.com/jacobsa/go-serial` for serial I/O
- ✅ handles all major NMEA sentence types (RMC, GGA, GSA, VTG, GSV)
- ✅ accumulates GSV messages across multiple sentences (MessageNumber/TotalMessages logic)
- ✅ publishes to 5 separate topics for granular data access
- ✅ configuration-driven serial port and MQTT settings

Future enhancements:

- implement time synchronization
- add data validation and outlier detection
- support additional sentence types (ZDA, GLL, etc.)

## 6. Consumers

### 6.1 Console MQTT subscriber (`cmd/console_mqtt`)

Entry point: `internal/app/RunConsoleMQTT()`

Responsibilities:

- read MQTT broker and topic configuration from `inertial_config.txt`
- connect to MQTT broker
- subscribe to all configured data streams
- decode JSON payloads into domain structs
- print formatted human-readable output to stdout

Output format:

```
[POSE]  ROLL=  20.45  PITCH=  -5.12  YAW= 123.67
[FUSE] ROLL=  20.45  PITCH=  -5.12  YAW= 123.67
[IMU-L] ax=    145 ay=   -230 az=  9850  gx=    10 gy=   -15 gz=    -5  mx=   -180 my=   210 mz=   -50
[IMU-R] ax=    150 ay=   -225 az=  9855  gx=    12 gy=   -18 gz=    -3  mx=   -175 my=   215 mz=   -48
[BMP-L] Temp=  23.45°C  Pressure= 101325 Pa (1013.25 mbar) (1013.25 hPa)
[BMP-R] Temp=  23.50°C  Pressure= 101330 Pa (1013.30 mbar) (1013.30 hPa)
[GPS ]  time=12:34:56 date=2025-12-09 lat=40.712776 lon=-74.005974 speed=5.2kn course=45.3° validity=A
```

Used primarily for:

- real-time monitoring and debugging
- validation during sensor integration
- headless operation without web UI
- ✅ Configuration-driven subscription topics

---

### 6.2 Web server (`cmd/web`)

Entry point: `internal/app/RunWeb()`

Responsibilities:

- read MQTT broker, topics, and web port from `inertial_config.txt`
- connect to MQTT broker
- subscribe to all configured data streams
- maintain in-memory cache of latest values per stream (protected by RWMutex)
- expose REST-style JSON APIs:

```
GET /api/orientation          → last Pose
GET /api/orientation/fused    → last fused Pose
GET /api/imu/left             → last left IMURaw
GET /api/imu/right            → last right IMURaw
GET /api/env/left             → last left Sample (temp + pressure)
GET /api/env/right            → last right Sample (temp + pressure)
GET /api/gps                  → last GPS Fix (full data)
GET /api/config               → system configuration (weather update interval, etc.)
```

- serve static HTML/JS dashboard from `web/` directory on configured port (default: 8080)
- ✅ Configuration-driven MQTT topics, broker address, and server port
- ✅ Config API endpoint for dynamic client configuration
- ✅ WebSocket endpoint for real-time calibration

Calibration endpoint:
```
WS /api/calibration/ws        → WebSocket for interactive calibration
```

Frontend behavior:

- polls most APIs every 500ms for real-time updates
- weather data cached and updated based on configurable interval (default 5 minutes)
- displays 10 dashboard cards in optimized compact layout:
  - Orientation (roll, pitch, yaw)
  - Fused orientation
  - GPS position and speed
  - Satellite sky plot (polar chart with 30°/60°/90° circles)
  - Satellite signal strength bar chart
  - Weather @ GPS (met.no API: temp, pressure, humidity, conditions)
  - Left and right IMU raw data
  - Left and right BMP environmental data
- ✅ Satellite visualizations with color-coded signal strength
- ✅ Weather integration with sea level pressure calculation
- ✅ Responsive grid layout optimized for single-screen viewing
- ✅ Dark theme with accent lighting
- ✅ Connection status for each stream
- ✅ **Interactive calibration UI** with 3D visualization (Three.js)
- ✅ Guided step-by-step calibration workflows

Calibration UI (`/calibration.html`):

- 3D device visualization showing required orientations
- Real-time WebSocket communication for calibration progress
- Three-phase guided workflow:
  1. **Gyroscope**: Static calibration + 3-axis dynamic rotations
  2. **Accelerometer**: 6-point orientation holds (±X, ±Y, ±Z)
  3. **Magnetometer**: Figure-8 motion for 20 seconds
- Live confidence metrics and progress tracking
- Timestamped JSON output with bias, scale, and offset values
- Automated state machine managing calibration flow

Future enhancements:

- Apply calibration coefficients in real-time sensor reads
- Calibration profile management (save/load/switch)
- Time-series graphs and data logging
- Configurable weather provider selection

---

### 6.3 Display consumer (`cmd/display`)

Entry point: `internal/app/RunDisplay()`

Responsibilities:

- read MQTT broker, topics, I2C addresses, and display configuration from `inertial_config.txt`
- initialize dual SSD1306 OLED displays (128x64 pixels) via I2C bus
- connect to MQTT broker and subscribe to topics based on display content configuration
- maintain in-memory cache of latest sensor data (protected by RWMutex)
- render display content at configured update intervals (default: 250ms)
- support configurable content per display

**Configuration parameters:**

```
DISPLAY_LEFT_I2C_ADDR=0x3C
DISPLAY_RIGHT_I2C_ADDR=0x3D
DISPLAY_UPDATE_INTERVAL=250
DISPLAY_LEFT_CONTENT=imu_raw_left
DISPLAY_RIGHT_CONTENT=imu_raw_right
```

**Content types:**

- `imu_raw_left` - Left IMU raw data (Accel X/Y/Z, Gyro X/Y/Z)
- `imu_raw_right` - Right IMU raw data (Accel X/Y/Z, Gyro X/Y/Z)
- `orientation_left` - Left orientation (Roll, Pitch, Yaw in degrees)
- `orientation_right` - Right orientation (Roll, Pitch, Yaw in degrees)
- `gps` - GPS position (Latitude, Longitude, Altitude)

**Display rendering:**

- 1-bit vertical LSB image format for SSD1306
- 7x13 bitmap font (golang.org/x/image/font/basicfont)
- Direct pixel buffer manipulation for performance
- Splash screens on startup

**Hardware requirements:**

- Two SSD1306 128x64 OLED displays
- I2C bus access (typically `/dev/i2c-1` on Raspberry Pi)
- Different I2C addresses for each display (default: 0x3C and 0x3D)
- Root privileges for I2C hardware access

**Data flow:**

```
MQTT Topics → Display Consumer → I2C Bus → SSD1306 Displays
(JSON)         (subscribe)        (periph.io)  (pixel data)
```

Design principles:

- **Configurable content**: Any display can show any data type
- **Independent operation**: Runs separately from web UI
- **Real-time updates**: Configurable refresh rate for responsiveness
- **Hardware abstraction**: periph.io for cross-platform I2C access

---

## 7. Calibration system

### 7.1 Overview

The Inertial Computer includes two calibration tools for IMU sensors:

1. **Web UI** (`/calibration.html`) - Interactive 3D-guided calibration
2. **CLI Tool** (`cmd/calibration`) - Console-based alternative

Both tools calibrate gyroscope, accelerometer, and magnetometer sensors independently for left and right IMUs.

### 7.2 Web UI calibration

**Architecture**:
- Frontend: `web/calibration.html` with Three.js 3D visualization
- Backend: `internal/app/calibration_handler.go` with WebSocket communication
- Protocol: Bidirectional JSON messages over WebSocket

**WebSocket message types** (client → server):
```json
{"action": "init", "imu": "left"}     // Initialize calibration
{"action": "next"}                     // Proceed to next step
{"action": "cancel"}                   // Cancel calibration
```

**WebSocket message types** (server → client):
```json
{"type": "phase", "phase": "gyro"}                    // Current phase
{"type": "step", "step": "gyro-x", "phase": "gyro"}  // Current step
{"type": "progress", "progress": 45.2}                // Progress %
{"type": "stats", "stats": {...}}                     // Live statistics
{"type": "action", "message": "ready"}                // Enable next button
{"type": "complete", "results": {...}}                // Calibration done
{"type": "error", "message": "..."}                   // Error occurred
```

**State machine** (`CalibrationSession`):
1. **Gyroscope phase** (4 steps):
   - Static calibration: Device on flat surface for 10 seconds
   - X-axis rotation: Pitch forward/backward
   - Y-axis rotation: Roll left/right
   - Z-axis rotation: Yaw/spin
   
2. **Accelerometer phase** (6 steps):
   - Six orientations: ±X, ±Y, ±Z axis pointing up
   - Hold each position for 5 seconds
   - Calculates bias and scale factors per axis
   
3. **Magnetometer phase** (1 step):
   - Figure-8 motion for 20 seconds
   - Covers all orientations for hard-iron offset
   - Diagonal soft-iron scale approximation

**Visualization**:
- Real-time 3D device model with animated orientations
- Color-coded axes (Red=X, Green=Y, Blue=Z)
- Automatic rotation animations showing required movements
- Progress bars and confidence metrics updated live

### 7.3 CLI calibration tool

**Location**: `cmd/calibration/main.go`

**Features**:
- Interactive console prompts with text-based guidance
- Same calibration algorithms as web UI
- L/R IMU selection with availability detection
- 5-second pause for user to read instructions
- Confidence scoring for each sensor type
- JSON output matching web UI format

**Usage**:
```bash
sudo ./calibration
```

**Output format** (`{imu}_{timestamp}_inertial_calibration.json`):
```json
{
  "version": 1,
  "imu": "left",
  "timestamp": "2025-12-23T12:34:56Z",
  "gyro_bias_x": -12.5,
  "gyro_bias_y": 8.3,
  "gyro_bias_z": -3.7,
  "gyro_confidence": 95.2,
  "accel_bias_x": 0.02,
  "accel_bias_y": -0.01,
  "accel_bias_z": 0.03,
  "accel_scale_x": 1.001,
  "accel_scale_y": 0.998,
  "accel_scale_z": 1.002,
  "accel_confidence": 92.5,
  "mag_offset_x": -180.5,
  "mag_offset_y": 210.3,
  "mag_offset_z": -50.2,
  "mag_scale_x": 1.05,
  "mag_scale_y": 0.98,
  "mag_scale_z": 1.02,
  "mag_confidence": 88.7,
  "total_samples": 450
}
```

### 7.4 Calibration algorithms

**Gyroscope**:
- Static bias: Mean of 100 samples with device stationary
- Dynamic refinement: Standard deviation during axis rotations
- Confidence: `100 / (1 + static_stddev * 1000)`

**Accelerometer**:
- Six-point calibration using gravity as reference (±1g)
- Bias: Center offset for each axis
- Scale: Deviation from expected 1g magnitude
- Confidence: `100 / (1 + avg_stddev * 100)`

**Magnetometer**:
- Hard-iron offset: Center of min/max ellipsoid
- Soft-iron scale: Diagonal approximation (avgRange / axisRange)
- Confidence: Based on axis range uniformity (minRange / maxRange * 100)

### 7.5 Integration (TODO)

Current status:
- ✅ Calibration tools functional and producing output
- ⚠️ Calibration coefficients not yet applied in producers
- ⚠️ No persistent profile management

Next steps:
1. Load calibration JSON files at producer startup
2. Apply corrections to sensor readings:
   - Gyro: `corrected = raw - bias`
   - Accel: `corrected = (raw - bias) * scale`
   - Mag: `corrected = (raw - offset) * scale`
3. Implement profile selection (default, custom, per-session)
4. Add calibration status indicators to web UI

---

## 8. Fusion strategy (current and future)

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

## 8. Deployment topology

### Single-machine setup (development/testing)

```
┌─────────────────────────────────┐
│ Raspberry Pi                    │
├─────────────────────────────────┤
│ [producer]  [gps_producer]      │ ← cmd/* entry points
│      │             │            │   (read inertial_config.txt)
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
- Mosquitto broker runs locally (default: port 1883, configurable)
- Web UI accessible via browser on same network (default: port 8080, configurable)
- Serial GPS on `/dev/serial0` (or configured serial port)
- All settings read from `inertial_config.txt` at startup

### Hardware connections

- **Left IMU (MPU9250)**: Configurable SPI device (default: `/dev/spidev6.0`), configurable CS pin (default: GPIO 18)
- **Right IMU (MPU9250)**: Configurable SPI device (default: `/dev/spidev0.0`), configurable CS pin (default: GPIO 8)
- **Left BMP sensor**: Configurable SPI device (default: `/dev/spidev6.1`)
- **Right BMP sensor**: Configurable SPI device (default: `/dev/spidev0.1`)
- **GPS module**: Configurable serial port (default: `/dev/serial0`), configurable baud rate (default: 9600)

---

## 9. Key architectural decisions

- **MQTT** chosen over direct RPC to simplify fan-out and allow independent restart of components
- **JSON** used for transport to maximize debuggability and enable inspection/logging
- **Configuration file** (`inertial_config.txt`) centralizes all deployment-specific settings
- **Singleton pattern** for hardware managers ensures efficient persistent access without re-initialization
- **Strict layering** enforces separation between domain models, transport, and hardware:
  - Domain structs (`internal/{orientation,imu,env,gps}`) have no dependencies
  - Sensor layer (`internal/sensors`) isolated from app logic with manager singletons
  - Producers/consumers (`internal/app`) never directly access hardware
  - Configuration layer (`internal/config`) provides global settings access
- **`internal/` package** used to prevent external consumers from coupling to implementation details
- **Pull-based REST API** for web UI instead of WebSockets (simpler, works behind proxies)

---

## 10. Extension ideas

- **Data logging + replay mode** for offline analysis and algorithm testing
- **WebSocket streaming** for real-time low-latency updates
- **3D visualization** (three.js) for intuitive orientation display
- **Time-series graphs** for IMU/BMP trends
- **Stream health monitoring** (staleness detection, error counters)
- **Additional sensors**: airspeed, wheel encoders, barometer, magnetometer standalone
- **Multi-producer coordination**: synchronized reads from multiple IMUs
- **Calibration tools**: magnetometer soft-iron/hard-iron compensation, IMU factory calibration
- **Cloud export**: optional MQTT bridge to cloud services for long-term analysis
- **CLI flag overrides**: command-line arguments to override config file values for debugging

---

## 11. Guiding principles

> Add sensors freely.
> Fuse intelligently.
> Consume everywhere.
> Never couple producers to consumers.
> Configure, don't hardcode.

---

## 12. Known limitations & TODOs

- **Left IMU**: Reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- **Right IMU**: Reads accel/gyro/mag from real MPU9250 via SPI; magnetometer reads working but not yet calibrated or fused into yaw
- **Magnetometer calibration**: Hard-iron and soft-iron correction not yet implemented
- **Yaw calculation**: Currently hardcoded to 0; needs gyro integration and magnetometer fusion
- **Environmental sensors**: ✅ Both left and right BMP sensors fully operational with bmxx80 driver (temperature and pressure in Pa, mbar, hPa)
  - ✅ Configurable oversampling, IIR filtering, standby time, and operating mode
  - ✅ Independent configuration for left and right sensors via `inertial_config.txt`
  - ✅ Default settings optimized for high accuracy: 16x pressure, 2x temperature, F8 filter, 62.5ms standby
- **Configuration**: ✅ All hardcoded values externalized to `inertial_config.txt`
- **Hardware management**: ✅ Singleton pattern implemented for IMU and BMP sensors

---

## 13. Recent changes

### IMU Refactoring (IMU_Refactor branch, merged to main)

**Configuration System** (`internal/config/config.go`, `inertial_config.txt`):
- Centralized all system settings in `inertial_config.txt`
- Simple KEY=VALUE format with comments
- Includes MQTT broker, client IDs, all topics, hardware paths, timing intervals, web server port
- Validation at startup ensures required fields are present
- All apps call `config.InitGlobal()` before running

**IMU Manager Pattern** (`internal/sensors/imu.go`):
- Created `IMUManager` singleton with `sync.RWMutex` for thread safety
- `GetIMUManager()` returns singleton instance
- `Init()` initializes both left and right IMU hardware once
- `ReadLeftIMU()` / `ReadRightIMU()` methods for sensor access
- `IsLeftIMUAvailable()` / `IsRightIMUAvailable()` for status checks
- Hardware persists across reads, no re-initialization overhead

**BMP Sensor Integration** (`internal/sensors/env.go`):
- Replaced stubs with real bmxx80 SPI driver
- `initBMP()` singleton with `sync.Once` initializes both sensors
- Opens SPI buses from config: `BMP_LEFT_SPI_DEVICE`, `BMP_RIGHT_SPI_DEVICE`
- Configurable sensor parameters for each BMP:
  - **Pressure oversampling** (`BMP_LEFT_PRESSURE_OSR` / `BMP_RIGHT_PRESSURE_OSR`): 0=off, 1=1x, 2=2x, 3=4x, 4=8x, 5=16x
  - **Temperature oversampling** (`BMP_LEFT_TEMP_OSR` / `BMP_RIGHT_TEMP_OSR`): 0=off, 1=1x, 2=2x, 3=4x, 4=8x, 5=16x
  - **Operating mode** (`BMP_LEFT_MODE` / `BMP_RIGHT_MODE`): 0=Sleep, 1=Forced, 3=Normal
  - **IIR filter** (`BMP_LEFT_IIR_FILTER` / `BMP_RIGHT_IIR_FILTER`): 0=off, 1=F2, 2=F4, 3=F8, 4=F16
  - **Standby time** (`BMP_LEFT_STANDBY_TIME` / `BMP_RIGHT_STANDBY_TIME`): 0=0.5ms, 1=62.5ms, 2=125ms, 3=250ms, 4=500ms, 5=1000ms, 6=2000ms, 7=4000ms
- Default configuration: 16x pressure oversampling, 2x temperature oversampling, Normal mode, F8 filter, 62.5ms standby (optimized for accuracy)
- `standbyTimeToDuration()` helper converts config values to Go `time.Duration`
- `ReadLeftEnv()` / `ReadRightEnv()` call `bmxx80.Sense()` with custom options
- Pressure conversion: `float64(e.Pressure) / float64(physic.Pascal)` for accurate values
- Returns temperature (°C) and pressure in Pa, mbar, and hPa

**Producer Updates** (`internal/app/imu_producer.go`):
- Uses `config.Get()` for all settings
- Uses `GetIMUManager()` for persistent IMU access
- Configurable sample interval via `IMU_SAMPLE_INTERVAL`
- Configurable console logging via `CONSOLE_LOG_INTERVAL`
- Logs all sensor data: pose, left IMU, right IMU, left BMP, right BMP once per configured interval
- Published right magnetometer data to `inertial/mag/right` topic

**Dependencies** (`go.mod`):
- Uses periph.io v3 packages for hardware access
- No local fork required; magnetometer support in upstream periph.io

**Next steps**:
1. Implement hard-iron and soft-iron calibration routines for magnetometers
2. Apply calibration to raw magnetometer readings
3. Integrate calibrated magnetometer data into yaw calculation
4. Implement gyro integration for dynamic yaw estimation
5. Add complementary filter or EKF for sensor fusion
6. Dual-IMU fusion for improved accuracy and fault detection

---

## 14. Magnetometer integration (test/debug mode)

### Driver changes (`internal/sensors/imu_source.go`):
- Added `InitMag()` call during IMU initialization for both left and right sensors
- Magnetometer initialization is non-fatal; system continues if mag unavailable
- Added `magCal` field to store magnetometer calibration parameters
- Added `magReady` flag to track whether magnetometer is operational
- `ReadRaw()` now calls `imu.ReadMag(magCal)` to retrieve magnetometer data
- Raw magnetometer values scaled as int16 (µT × 10) for consistency with accel/gyro
- Overflow detection implemented; overflows are silently skipped

### Producer magnetometer publishing (`internal/app/imu_producer.go`):
- Added `magNorm()` helper function to compute magnetic field magnitude
- MQTT topics `inertial/mag/left` and `inertial/mag/right` publish magnetometer-only data with:
  - Raw mx, my, mz values
  - Computed field magnitude (|B|)
  - RFC3339 timestamp
- Updated logging to include magnetometer readings: `mx=X my=Y mz=Z |B|=N`
- Enables validation of magnetometer behavior before fusion integration

See [TODO.md](TODO.md) for detailed prioritized task list.

