# Inertial Computer - High-Level Repository Overview

**Quick Summary**: Developer-oriented inertial sensing platform built in Go for Raspberry Pi, using MQTT message-bus architecture to read multiple hardware sensors (dual IMUs, magnetometers, environmental sensors, GPS) and distribute data to multiple consumers (console, web UI, OLED displays).

---

## 🎯 What This Project Does

This is a **multi-sensor data acquisition and visualization system** designed for embedded Linux (Raspberry Pi) that:

1. **Reads multiple hardware sensors** simultaneously:
   - Dual MPU9250 IMUs (accelerometer + gyroscope + magnetometer) - left and right
   - Dual BMP280/BMP388 environmental sensors (temperature + pressure)
   - GPS module with full NMEA parsing (position, velocity, satellites)
   - Weather data from met.no API based on GPS location

2. **Publishes all sensor data** over MQTT message bus:
   - Raw sensor readings (accelerometer, gyroscope, magnetometer)
   - Computed orientation (roll, pitch, yaw)
   - GPS data (position, velocity, quality, satellites)
   - Environmental data (temperature, pressure)
   - Weather information (external API)

3. **Provides multiple data consumers**:
   - Console MQTT subscriber for debugging
   - Web server with real-time dashboard UI
   - Dual OLED displays (SSD1306) with configurable content
   - Calibration tools (web-based and CLI)
   - Hardware register debugger for MPU9250

---

## 🏗️ System Architecture

### Message Bus Pattern (MQTT-Based)

```
┌─────────────────────────────────────────────────────────────┐
│                      MQTT Broker (Mosquitto)                 │
│                    tcp://localhost:1883                      │
└────┬────────────────────────────────────────────────────┬───┘
     │                                                      │
     │ Publishers                                           │ Subscribers
     │                                                      │
┌────▼──────────────┐    ┌──────────────────┐    ┌────────▼─────────┐
│ IMU Producer      │    │  GPS Producer     │    │ Console MQTT     │
│ (cmd/imu_producer)│    │ (cmd/gps_producer)│    │ (cmd/console_mqtt│
│                   │    │                   │    │                  │
│ Publishes:        │    │ Publishes:        │    │ Subscribes:      │
│ • inertial/imu/*  │    │ • inertial/gps/*  │    │ • inertial/#     │
│ • inertial/pose/* │    │                   │    └──────────────────┘
│ • inertial/bmp/*  │    │                   │
└───────────────────┘    └──────────────────┘    ┌──────────────────┐
                                                  │ Web Server       │
                                                  │ (cmd/web)        │
                                                  │                  │
                                                  │ REST API + UI    │
                                                  │ Port: 8080       │
                                                  └──────────────────┘

                                                  ┌──────────────────┐
                                                  │ Display Consumer │
                                                  │ (cmd/display)    │
                                                  │                  │
                                                  │ Dual OLED (I2C)  │
                                                  └──────────────────┘
```

**Key Design Principle**: Complete decoupling - producers and consumers never directly communicate. All data flows through MQTT topics.

---

## 📦 Repository Structure

```
inertial_computer/
├── cmd/                          # Executable programs (main packages)
│   ├── imu_producer/            # Reads IMU sensors, publishes to MQTT
│   ├── gps_producer/            # Reads GPS, publishes to MQTT
│   ├── console_mqtt/            # Console subscriber for debugging
│   ├── web/                     # Web server + REST API
│   ├── display/                 # Dual OLED display consumer
│   ├── calibration/             # CLI calibration tool
│   └── register_debug/          # MPU9250 hardware register debugger
│
├── internal/                     # Internal packages (not importable)
│   ├── app/                     # Application logic for each program
│   ├── config/                  # Configuration system
│   ├── sensors/                 # Hardware abstraction layer (HAL)
│   ├── imu/                     # IMU data models
│   ├── gps/                     # GPS data models
│   ├── orientation/             # Orientation computation (pose)
│   └── env/                     # Environmental sensor models
│
├── web/                         # Static web assets
│   ├── index.html              # Main dashboard
│   ├── calibration.html        # Interactive calibration UI
│   └── register_debug.html     # Hardware register debugger UI
│
├── scripts/                     # Helper scripts
│
├── inertial_config.txt         # Central configuration file (KEY=VALUE)
│
└── Documentation
    ├── README.md               # Quick introduction
    ├── ARCHITECTURE.md         # Detailed system design (983 lines)
    ├── QUICKSTART.md           # Setup and build instructions
    ├── TODO.md                 # Current status and task list
    ├── CALIBRATION_UI.md       # Calibration tool documentation
    └── USEFUL_COMMANDS.md      # Common development commands
```

---

## 🔧 Technology Stack

- **Language**: Go 1.25.5
- **Message Bus**: MQTT (Eclipse Paho client, Mosquitto broker)
- **Hardware Access**: periph.io (custom fork with magnetometer + OLED enhancements)
- **GPS Parsing**: go-nmea library
- **Web Framework**: Standard library (net/http) + Gorilla WebSocket
- **Frontend**: Vanilla JavaScript with Chart.js and Three.js
- **Target Platform**: Raspberry Pi (embedded Linux)
- **Hardware Interfaces**: SPI, I2C, GPIO, UART (serial)

---

## 🎛️ Key Features & Capabilities

### ✅ Fully Operational
- **Dual IMU System**: Left and right MPU9250 sensors with 9-axis data (accel, gyro, mag)
- **GPS Integration**: Full NMEA sentence parsing (RMC, GGA, GSA, VTG, GSV)
- **Satellite Tracking**: Real-time visualization with sky plot and signal strength
- **Environmental Sensing**: Temperature and pressure from dual BMP sensors
- **Weather Integration**: External API (met.no) with GPS-based location
- **Real-time Dashboard**: Web UI with all sensor data and visualizations
- **OLED Displays**: Dual configurable displays showing IMU, orientation, or GPS data
- **Calibration Tools**: Both web-based (3D interactive) and CLI-based workflows
- **Register Debugger**: Low-level MPU9250 hardware debugging tool
- **Configuration System**: All settings externalized to `inertial_config.txt`

### ⚠️ In Progress
- **Magnetometer Calibration**: Framework exists, applying coefficients to readings
- **Sensor Fusion**: Integrating gyro + mag for accurate yaw calculation
- **Testing Infrastructure**: Unit and integration tests

---

## 📊 Data Flow Example

1. **IMU Producer** reads raw sensor values from MPU9250 via SPI
2. Computes orientation (roll/pitch) from accelerometer
3. Publishes to MQTT:
   - Raw data → `inertial/imu/left`
   - Orientation → `inertial/pose/left`
4. **Console MQTT** subscribes to `inertial/#` and logs all data
5. **Web Server** caches latest data from all topics, serves REST API
6. **Browser** polls REST API every 500ms, updates charts in real-time
7. **Display Consumer** subscribes to topics, renders to OLED screens

**Result**: Any component can restart independently without affecting others.

---

## ⚙️ Configuration Philosophy

**Critical Rule**: NEVER hardcode anything configurable.

All runtime settings live in `inertial_config.txt`:
- MQTT broker addresses and topics
- Hardware device paths (SPI, I2C, serial)
- Sensor ranges and sample rates
- Pin assignments (GPIO, CS)
- Timing intervals
- Port numbers

Example:
```ini
MQTT_BROKER=tcp://localhost:1883
IMU_LEFT_SPI_DEVICE=/dev/spidev6.0
IMU_LEFT_CS_PIN=18
IMU_SAMPLE_INTERVAL=100
WEB_SERVER_PORT=8080
GPS_SERIAL_PORT=/dev/serial0
```

**Access Pattern**:
```go
config.InitGlobal("./inertial_config.txt")  // First line in main()
broker := config.Get().MQTTBroker           // Access throughout app
```

---

## 🔬 Hardware Abstraction

**Key Boundary**: Only `internal/sensors/` imports periph.io.

- **Singleton Pattern**: `IMUManager` maintains persistent hardware connections
- **Thread-Safe**: `sync.RWMutex` for concurrent reads
- **Graceful Degradation**: Individual sensor failures don't crash system
- **Pure Functions**: Orientation computation is hardware-independent

Example:
```go
// Hardware layer (internal/sensors/imu.go)
manager := GetIMUManager()
manager.Init()  // Once per program
raw := manager.ReadRaw("left")

// Pure computation (internal/orientation/orientation.go)
pose := AccelToPose(raw.Ax, raw.Ay, raw.Az)
```

---

## 🚀 Getting Started

### Prerequisites
1. Raspberry Pi with enabled interfaces (SPI, I2C, UART)
2. Go 1.25.5+ installed
3. Mosquitto MQTT broker running
4. Hardware sensors connected

### Quick Start
```bash
# 1. Install custom periph.io fork (required for magnetometer + OLED)
git clone https://github.com/relabs-tech/devices.git
# Update go.mod replace directive

# 2. Build all components
go build -o imu_producer ./cmd/imu_producer/
go build -o gps_producer ./cmd/gps_producer/
go build -o web ./cmd/web/
go build -o console_mqtt ./cmd/console_mqtt/
go build -o display ./cmd/display/

# 3. Run (producers need sudo for hardware access)
sudo ./imu_producer
sudo ./gps_producer
./web
./console_mqtt
sudo ./display

# 4. Access web UI
# Open browser to http://raspberry-pi-ip:8080
```

See [QUICKSTART.md](QUICKSTART.md) for detailed instructions.

---

## 🐛 Development Workflow

### Building
```bash
go build -o imu_producer ./cmd/imu_producer/
```

### Testing Hardware
- Mock mode available in producers (toggle `useMock` flag)
- No real hardware needed for development

### MQTT Debugging
```bash
# Subscribe to all topics
mosquitto_sub -h localhost -t 'inertial/#' -v

# Test specific topic
mosquitto_sub -h localhost -t 'inertial/imu/left'
```

### Common Commands
See [USEFUL_COMMANDS.md](USEFUL_COMMANDS.md) for comprehensive list.

---

## 📖 Documentation Deep Dive

- **[ARCHITECTURE.md](ARCHITECTURE.md)** (983 lines): Complete system design, data models, fusion algorithms, MQTT topics
- **[README.md](README.md)**: Project introduction, current status, hardware setup
- **[TODO.md](TODO.md)**: Task tracking, known issues, next steps
- **[QUICKSTART.md](QUICKSTART.md)**: Build instructions, MQTT setup, running components
- **[CALIBRATION_UI.md](CALIBRATION_UI.md)**: Calibration tool usage (web + CLI)

---

## 🎯 Project Status Summary

**Maturity**: Active development, core functionality complete, calibration/fusion in progress

**Production Readiness**: 
- ✅ Data acquisition: Production-ready
- ✅ MQTT architecture: Production-ready
- ✅ Web UI: Production-ready
- ⚠️ Calibration: Framework complete, needs validation
- ⚠️ Sensor fusion: In development

**Known Limitations**:
- Yaw currently placeholder (hardcoded to 0) until mag fusion implemented
- Calibration coefficients not yet applied to sensor readings
- Limited automated testing

---

## 🔑 Key Design Patterns to Remember

1. **Message Bus Isolation**: Never add direct dependencies between producers/consumers
2. **Configuration First**: All configurable values in `inertial_config.txt`
3. **Hardware Abstraction**: Only `internal/sensors/` imports periph.io
4. **Graceful Degradation**: Individual sensor failures are non-fatal
5. **Singleton Hardware**: `IMUManager` maintains persistent connections
6. **Pure Functions**: Orientation computation is hardware-independent

---

## 🤝 Contributing

When adding new features:
1. Add configuration to `inertial_config.txt` first
2. Update `internal/config/config.go` struct
3. Never hardcode hardware paths, pins, topics, or timing
4. Keep hardware access in `internal/sensors/`
5. Use MQTT for all component communication
6. Follow existing error handling patterns (log warnings, don't crash)

---

## 📝 License

MIT License - Copyright © 2026 Daniel Alarcon Rubio / Relabs Tech

See [LICENSE](LICENSE) file for details.

---

## 📞 Quick Reference

- **Author**: Daniel Alarcon Rubio ([@relabs-tech](https://github.com/relabs-tech))
- **Language**: Go 1.25.5
- **Target**: Raspberry Pi (embedded Linux)
- **Architecture**: MQTT message bus
- **Hardware**: MPU9250 (IMU), BMP280/388 (env), GPS (NMEA), SSD1306 (OLED)
- **Web UI**: http://localhost:8080
- **Config**: `inertial_config.txt`

---

*For detailed technical documentation, see [ARCHITECTURE.md](ARCHITECTURE.md). For setup instructions, see [QUICKSTART.md](QUICKSTART.md).*
