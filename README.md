# Inertial Computer Project

## Overview

The **Inertial Computer** project runs on a Raspberry Pi and integrates:

- inertial orientation data (IMU)
- GPS data (NMEA)

Data is published via **MQTT** and consumed by multiple independent sinks, including:

- a terminal-based console subscriber
- a web-based UI served directly from the Pi

The architecture is designed to be:

- modular and extensible
- hardware-agnostic at the application layer
- easy to debug and evolve over time

---

## What This Repository Contains

This repository contains several Go executables (in `cmd/`) and shared internal packages (in `internal/`). Each executable has a single responsibility, and components communicate via MQTT.

For a **detailed technical breakdown of the architecture**, data flow, and design decisions, see:

➡ **ARCHITECTURE.md**

---

## Main Components

### Producers

- **Orientation Producer** (`cmd/producer`)
  - Reads orientation from a mock source (or real IMU later)
  - Publishes `Pose` messages to MQTT topic `inertial/pose`

- **GPS Producer** (`cmd/gps_producer`)
  - Reads NMEA sentences from a serial GPS
  - Publishes `Fix` messages to MQTT topic `inertial/gps`

### Consumers

- **Web Server** (`cmd/web`)
  - Subscribes to both `inertial/pose` and `inertial/gps`
  - Exposes `/api/orientation` and `/api/gps`
  - Serves a web UI showing orientation and GPS panels

- **Console Subscriber** (`cmd/console_mqtt`)
  - Subscribes to both MQTT topics
  - Prints orientation and GPS data to the terminal

---

## MQTT Topics

| Topic          | Payload | Description        |
|---------------|---------|--------------------|
| inertial/pose | Pose    | Orientation data   |
| inertial/gps  | Fix     | GPS fix data       |

---

## Running the System (Typical)

On the Raspberry Pi, in separate terminals:

```bash
# Orientation producer (mock or real IMU)
go run ./cmd/producer

# GPS producer (NMEA → MQTT)
go run ./cmd/gps_producer

# Web server (MQTT → HTTP UI)
go run ./cmd/web

# Console subscriber (MQTT → terminal)
go run ./cmd/console_mqtt
```

Then open a browser on your Mac:

```text
http://<pi-hostname-or-ip>:8080/
```

---

## Project Status

- ✅ Mock orientation pipeline working
- ✅ GPS data published via MQTT
- ✅ Web UI shows orientation + GPS
- ✅ Console subscriber prints orientation + GPS

Next steps include swapping in real hardware sources and adding richer visualisation.
