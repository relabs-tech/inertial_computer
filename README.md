# Inertial Computer Project

## Overview

This project runs on a **Raspberry Pi** and integrates real-time inertial (IMU) and GPS data, publishes it over **MQTT**, and exposes multiple independent *sinks* such as:

- terminal console output
- a web-based UI served from the Pi
- future sinks (logging, recording, replay, analytics, etc.)

The system is designed so that:

- hardware access is isolated
- data transport is decoupled via MQTT
- consumers are independent and replaceable

---

## High-Level Architecture

```
┌────────────┐
│  IMU / GPS │   (real HW later, mock today)
└─────┬──────┘
      │
      ▼
┌────────────┐
│ producers  │  cmd/producer, cmd/gps_producer
│ (Go apps)  │
└─────┬──────┘
      │  MQTT JSON
      ▼
┌────────────┐
│ Mosquitto  │  tcp://localhost:1883
│  Broker    │
└─────┬──────┘
      │
      ├──────────────┐
      ▼              ▼
┌────────────┐  ┌────────────┐
│ console    │  │ web server │
│ subscriber │  │ + browser  │
└────────────┘  └────────────┘
```

**MQTT is the central event bus.**

- Producers do not know who consumes data
- Consumers do not know how data is produced
- Components can restart independently

---

## Repository Structure

```
inertial_computer/
├── cmd/
│   ├── producer/        # publishes orientation (IMU / mock) to MQTT
│   ├── gps_producer/    # publishes GPS fixes (NMEA) to MQTT
│   ├── console/         # direct console (non-MQTT, legacy / test)
│   ├── console_mqtt/    # console subscriber via MQTT
│   └── web/             # web server (MQTT subscriber + HTTP UI)
│
├── internal/
│   ├── app/             # application logic (RunWeb, RunConsoleMQTT, etc.)
│   ├── orientation/     # orientation domain (Pose, Source interface)
│   └── gps/             # GPS domain (Fix struct)
│
├── web/                 # static web UI (HTML/CSS/JS)
├── go.mod
├── go.sum
└── README.md
```

Conventions:
- `cmd/` contains **executables only**
- `internal/` contains **non-importable application code**
- domain packages contain *pure data models*

---

## Core Data Models

### Orientation (IMU)

```go
type Pose struct {
    Roll  float64 `json:"roll"`
    Pitch float64 `json:"pitch"`
    Yaw   float64 `json:"yaw"`
}
```

- immutable orientation snapshot
- published to MQTT topic:

```text
inertial/pose
```

---

### GPS

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

- combined GPS fix derived from NMEA
- published to MQTT topic:

```text
inertial/gps
```

---

## Orientation Source Abstraction

Orientation data is abstracted using an interface:

```go
type Source interface {
    Next() (Pose, error)
}
```

Implementations:
- mock source (sin/cos based)
- future IMU-backed source (periph.io)

This allows mock and real hardware to be swapped without changing consumers.

---

## Producers

### Orientation Producer (`cmd/producer`)

Responsibilities:
- create an `orientation.Source`
- periodically call `Next()`
- JSON-encode `Pose`
- publish to MQTT

Owns **hardware access** and **timing**.


### GPS Producer (`cmd/gps_producer`)

Responsibilities:
- read NMEA sentences from serial
- parse sentences
- build `gps.Fix`
- publish to MQTT

No knowledge of consumers or UI.

---

## MQTT Topics

| Topic         | Payload | Description       |
|--------------|---------|-------------------|
| inertial/pose | Pose    | orientation data  |
| inertial/gps  | Fix     | GPS data          |

---

## Web Server (`cmd/web`)

Responsibilities:
- subscribe to MQTT topics
- store latest Pose and Fix
- expose JSON APIs
- serve static UI

Endpoints:

```text
/api/orientation   → Pose (JSON)
/api/gps           → Fix (JSON)
```

---

## Console Subscriber (`cmd/console_mqtt`)

Responsibilities:
- subscribe to MQTT
- print orientation and GPS data

Example output:

```text
[POSE] ROLL= 12.34 PITCH= -2.10 YAW=123.45
[GPS ] lat=40.123456 lon=-3.654321 speed=1.2kn course=87.0 validity=A
```

---

## Web UI

- single-page HTML UI
- polls `/api/orientation` and `/api/gps`
- displays two panels:
  - Orientation (Roll / Pitch / Yaw)
  - GPS (Lat / Lon / Speed / Course)

Future evolution:
- WebSocket streaming
- 3D orientation rendering

---

## Design Principles

- separation of concerns
- dependency inversion via interfaces
- immutable domain values
- explicit Go module dependency management
- one responsibility per executable
- MQTT as an event bus

---

## Next Steps

- plug in real IMU source
- fuse multiple GPS sentence types
- add logging / recording sink
- switch web UI to WebSockets
- add 3D visualization

---

## Key Takeaway

> Hardware is isolated.
> Data flows through MQTT.
> Sinks are independent.
> The system scales without architectural rewrites.
