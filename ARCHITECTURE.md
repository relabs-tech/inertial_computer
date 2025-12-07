# Inertial Computer – Architecture and Design

This document describes the internal architecture of the **Inertial Computer** project: data flow, package structure, core data models, and design principles.

---

## 1. Architectural Overview

The system is structured around **MQTT as a central event bus**.

```
┌────────────┐
│ IMU / GPS  │   (real HW later, mock today)
└─────┬──────┘
      │
      ▼
┌────────────┐
│ Producers  │  cmd/producer, cmd/gps_producer
│ (Go apps)  │
└─────┬──────┘
      │  MQTT (JSON)
      ▼
┌────────────┐
│ Mosquitto  │  tcp://localhost:1883
│  Broker    │
└─────┬──────┘
      │
      ├──────────────┐
      ▼              ▼
┌────────────┐  ┌────────────┐
│ Console    │  │ Web Server │
│ Subscriber │  │ + Browser  │
└────────────┘  └────────────┘
```

Key properties:

- producers do not know consumers
- consumers do not know producers
- components can restart independently

---

## 2. Repository Structure

```
inertial_computer/
├── cmd/
│   ├── producer/        # Orientation producer (IMU or mock)
│   ├── gps_producer/    # GPS (NMEA) producer
│   ├── console/         # direct console (legacy / testing)
│   ├── console_mqtt/    # MQTT console subscriber
│   └── web/             # Web server (MQTT subscriber + UI)
│
├── internal/
│   ├── app/             # application orchestration logic
│   ├── orientation/     # orientation domain (Pose, Source)
│   └── gps/             # GPS domain (Fix)
│
├── web/                 # static web UI
├── go.mod
├── go.sum
```

Rules applied:
- one executable per folder under `cmd/`
- `internal/` packages cannot be imported externally
- domain packages contain no transport or UI code

---

## 3. Core Domain Models

### 3.1 Orientation

```go
type Pose struct {
    Roll  float64 `json:"roll"`
    Pitch float64 `json:"pitch"`
    Yaw   float64 `json:"yaw"`
}
```

Published on MQTT topic:

```
inertial/pose
```

---

### 3.2 GPS

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

Published on MQTT topic:

```
inertial/gps
```

---

## 4. Orientation Source Abstraction

Orientation data is produced through an interface:

```go
type Source interface {
    Next() (Pose, error)
}
```

Implementations:

- `mockSource` – deterministic mock for testing
- `imuSource` – future implementation backed by periph.io

This allows hardware and application logic to evolve independently.

---

## 5. Producers

### Orientation Producer

- owns timing
- accesses IMU or mock source
- publishes Pose JSON to MQTT

### GPS Producer

- reads NMEA from serial port
- parses sentences
- constructs Fix values
- publishes Fix JSON to MQTT

Neither producer has any knowledge of consumers.

---

## 6. Consumers

### Web Server

Responsibilities:

- subscribe to `inertial/pose` and `inertial/gps`
- store latest Pose and Fix
- expose REST endpoints:

```
/api/orientation
/api/gps
```

- serve static web UI

### Console Subscriber

Responsibilities:

- subscribe to both MQTT topics
- decode JSON payloads
- print human-readable output

---

## 7. Web UI

The UI is a static HTML application served by the Go web server.

Features:

- orientation panel (roll / pitch / yaw)
- GPS panel (lat / lon / speed / course / time / date)
- regular polling of REST endpoints

Future upgrades:

- WebSocket push model
- 3D orientation rendering

---

## 8. Design Principles

- separation of concerns
- explicit data ownership
- interface-driven design
- immutable domain values
- MQTT as a message bus
- multiple independent sinks

---

## 9. Evolution Path

Natural next steps:

- replace mock orientation with real IMU source
- fuse multiple GPS sentence types (RMC + GGA)
- add persistent logging / recording
- introduce playback / simulation mode
- switch web polling to WebSockets
- add 3D visualisation

---

## 10. Key Takeaway

> Hardware is isolated.
> Data flows through MQTT.
> Consumers are independent.
> The system scales without architectural rewrites.

