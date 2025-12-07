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

    Ax int16 `json:"ax"`
    Ay int16 `json:"ay"`
    Az int16 `json:"az"`

    Gx int16 `json:"gx"`
    Gy int16 `json:"gy"`
    Gz int16 `json:"gz"`

    Mx int16 `json:"mx"`
    My int16 `json:"my"`
    Mz int16 `json:"mz"`
}
```

Published on:

- `inertial/imu/left`
- `inertial/imu/right`

---

### 2.3 Environmental data (BMP)

```go
type Sample struct {
    Source string  `json:"source"` // "left" | "right"
    Temperature float64 `json:"temp_c"`
    Pressure    float64 `json:"pressure_pa"`
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

## 3. Hardware abstraction

All direct sensor access is isolated in:

```
internal/sensors/
```

This layer is responsible for:

- talking to periph.io drivers
- handling I2C/SPI details
- converting driver output into domain structs

The rest of the system **never** imports periph or hardware-specific code.

---

## 4. Producers

### 4.1 Main sensor producer (`cmd/producer`)

Responsibilities:

- read left/right IMU data (accel + gyro + mag)
- read left/right BMP data
- compute orientation estimates
- publish raw + fused data to MQTT

The current fused pose may be derived from a single IMU; future work will introduce true multi-sensor fusion.

---

### 4.2 GPS producer (`cmd/gps_producer`)

Responsibilities:

- read NMEA sentences from serial port
- parse RMC (and later GGA) messages
- publish GPS fixes to MQTT

---

## 5. Consumers

### 5.1 Console subscriber (`cmd/console_mqtt`)

- subscribes to all MQTT topics
- decodes JSON payloads
- prints structured human-readable output

Used primarily for:

- debugging
- validation during sensor integration
- headless operation

---

### 5.2 Web server (`cmd/web`)

Responsibilities:

- subscribe to MQTT topics
- store latest values per stream
- expose REST-style APIs:

```text
/api/orientation
/api/orientation/fused
/api/imu/left
/api/imu/right
/api/env/left
/api/env/right
/api/gps
```

- serve a static HTML/JS dashboard

The UI polls these APIs periodically. WebSockets are a possible future enhancement.

---

## 6. Fusion strategy (current and future)

Current state:

- fused pose == left IMU-derived pose
- magnetometer data is published but not yet used for yaw stabilization

Planned evolution:

- gyro integration + accelerometer correction
- magnetometer-based yaw alignment
- left/right IMU cross-checking
- optional extended Kalman filter (EKF)

Fusion will remain internal to the producer and will not affect consumers.

---

## 7. Key architectural decisions

- MQTT chosen over direct RPC to simplify fan-out
- JSON used for debuggability and inspection
- strict separation between domain, transport, and hardware
- `internal/` used to prevent external coupling

---

## 8. Extension ideas

- data logging + replay mode
- offline analysis tooling
- WebSocket streaming
- 3D visualization (three.js)
- additional sensors (e.g. airspeed, wheel encoders)

---

## 9. Guiding principle

> Add sensors freely.
> Fuse intelligently.
> Consume everywhere.
> Never couple producers to consumers.

