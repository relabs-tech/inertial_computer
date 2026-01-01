# Useful Console Commands

## Process Management

### Check Running Processes
```bash
# Show all 'go run' processes with full command line
pgrep -af "go run"

# Check for specific program (web server or IMU producer)
ps aux | grep -E '[w]eb|[i]mu_producer'

# Show background jobs in current terminal session
jobs

# Kill a specific process by PID
kill <PID>

# Kill all Go processes (use with caution)
pkill -f "go run"
```

### Check Hardware Access
```bash
# Check which process is using SPI devices (for hardware conflicts)
lsof /dev/spidev*

# Check I2C device access
lsof /dev/i2c-*

# List available SPI devices
ls -la /dev/spidev*

# List available I2C devices
ls -la /dev/i2c-*
```

## Building and Running

### Build Components
```bash
# Build individual components
go build -o bin/imu_producer ./cmd/imu_producer/
go build -o bin/web ./cmd/web/
go build -o bin/gps_producer ./cmd/gps_producer/
go build -o bin/console_mqtt ./cmd/console_mqtt/
go build -o bin/display ./cmd/display/

# Build all at once
go build -o bin/ ./cmd/...
```

### Run Components
```bash
# Run IMU producer (requires sudo for hardware access)
sudo go run ./cmd/imu_producer/main.go
# or with built binary
sudo ./bin/imu_producer

# Run web server (no sudo needed when using --skip-imu flag)
go run ./cmd/web/main.go --skip-imu
# or with built binary
./bin/web --skip-imu

# Run web server with IMU access for calibration (requires sudo)
sudo go run ./cmd/web/main.go

# Run GPS producer (requires sudo for serial port access)
sudo go run ./cmd/gps_producer/main.go

# Run console MQTT subscriber (no sudo needed)
go run ./cmd/console_mqtt/main.go

# Run display controller (requires sudo for I2C/SPI access)
sudo go run ./cmd/display/main.go
```

## MQTT Debugging

### Subscribe to Topics
```bash
# Subscribe to all topics (requires mosquitto-clients package)
mosquitto_sub -h localhost -t 'inertial/#' -v

# Subscribe to specific topics
mosquitto_sub -h localhost -t 'inertial/imu/left'
mosquitto_sub -h localhost -t 'inertial/imu/right'
mosquitto_sub -h localhost -t 'inertial/pose/left'
mosquitto_sub -h localhost -t 'inertial/pose/right'
mosquitto_sub -h localhost -t 'inertial/pose/fused'
mosquitto_sub -h localhost -t 'inertial/gps'
mosquitto_sub -h localhost -t 'inertial/bmp/left'
mosquitto_sub -h localhost -t 'inertial/bmp/right'
```

### Publish Test Messages
```bash
# Test pose message
mosquitto_pub -h localhost -t 'inertial/pose' -m '{"roll":12.5,"pitch":-5.3,"yaw":180.0}'

# Test IMU raw message
mosquitto_pub -h localhost -t 'inertial/imu/left' -m '{"source":"left","ax":100,"ay":200,"az":16384,"gx":10,"gy":20,"gz":30,"mx":250,"my":300,"mz":350}'
```

### MQTT Broker Management
```bash
# Check if MQTT broker is running
systemctl status mosquitto

# Start/stop/restart MQTT broker
sudo systemctl start mosquitto
sudo systemctl stop mosquitto
sudo systemctl restart mosquitto

# View MQTT broker logs
sudo journalctl -u mosquitto -f
```

## Configuration

### Edit Configuration
```bash
# Edit main configuration file
nano inertial_config.txt

# View current configuration
cat inertial_config.txt
```

## Git and Version Control

```bash
# Check current status
git status

# View recent changes
git diff

# View commit history
git log --oneline -10
```

## System Information

### Hardware Information
```bash
# Check Raspberry Pi model and memory
cat /proc/cpuinfo | grep Model
free -h

# Check available GPIO/I2C/SPI
ls -la /dev/gpiochip*
ls -la /dev/i2c-*
ls -la /dev/spidev*

# Check serial ports
ls -la /dev/ttyUSB* /dev/ttyACM* /dev/serial*
```

### Temperature and Performance
```bash
# CPU temperature
vcgencmd measure_temp

# CPU frequency
vcgencmd measure_clock arm
```

## Development Workflow

### Typical Startup Sequence
```bash
# Terminal 1: Start IMU producer (requires sudo)
sudo ./bin/imu_producer

# Terminal 2: Start GPS producer (requires sudo)
sudo ./bin/gps_producer

# Terminal 3: Start web server (no sudo with --skip-imu)
./bin/web --skip-imu

# Terminal 4: Start display controller (requires sudo)
sudo ./bin/display

# Terminal 5: Monitor MQTT messages
mosquitto_sub -h localhost -t 'inertial/#' -v
```

### Quick Test Without Building
```bash
# Test with all producers and web UI
sudo go run ./cmd/imu_producer/main.go &
sudo go run ./cmd/gps_producer/main.go &
go run ./cmd/web/main.go --skip-imu &
```

## Troubleshooting

### Permission Issues
```bash
# If you get permission errors with hardware, check user groups
groups

# Add user to necessary groups (requires logout/login after)
sudo usermod -a -G dialout,i2c,spi,gpio $USER

# Check device permissions
ls -la /dev/spidev* /dev/i2c-* /dev/ttyUSB*
```

### Port Conflicts
```bash
# Check what's using port 8080 (web server default)
sudo lsof -i :8080

# Kill process using port 8080
sudo fuser -k 8080/tcp
```

### Hardware Testing
```bash
# Test I2C devices (requires i2c-tools package)
sudo i2cdetect -y 1

# Test SPI loopback (if supported)
# This is hardware-specific
```
