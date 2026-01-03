# Quick Start Guide

This guide will get your Inertial Computer system up and running in minutes.

## Prerequisites

- Go 1.21 or later installed
- Raspberry Pi with MPU9250 IMUs connected via SPI
- GPS module connected via serial
- MQTT broker running (Mosquitto recommended)

## Step 0: Install Custom periph.io Devices Fork (Required for Magnetometer)

The project uses a custom fork of `periph.io/x/devices` for enhanced MPU9250 magnetometer support. Clone it to the expected location:

```bash
# Clone the custom devices fork to the expected directory
git clone https://github.com/relabs-tech/devices.git ~/go/src/github.com/relabs-tech/devices

# Verify the clone was successful
ls ~/go/src/github.com/relabs-tech/devices
# Should show: go.mod, go.sum, devices/, and other directories
```

**Why is this needed?**
The standard `periph.io/x/devices` doesn't have complete magnetometer (AK8963) support via MPU9250's internal I2C. The custom fork in `github.com/relabs-tech/devices` includes critical enhancements:

**Fork Enhancements over Standard periph.io:**
- **AK8963 Magnetometer Driver**: Direct support for reading AK8963 magnetometer via MPU9250's internal I2C master (EXT_SENS_DATA registers)
- **MagCal Calibration**: Calibration data structure for hard-iron offset and soft-iron scale factors (not available in standard)
- **InitMag() Method**: Initialize internal I2C master for magnetometer communication with automatic factory calibration loading
- **ReadMag() Method**: Read calibrated magnetometer values in µT (Tesla), accessible from any IMU instance
- **Overflow Detection**: Magnetometer overflow flag handling to detect when readings exceed sensor range
- **SetSpiTransport() Support**: Extended SPI transport layer with configurable read/write speeds for register debugging
- **RegisterRead/Write Methods**: Direct register access (0x00-0x7F) for all 128 MPU9250 registers via SPI (used by register debug tool)
- **GetRotationX/Y/Z() Methods**: Extended gyroscope access functions with individual axis queries

The `inertial_computer` project's `go.mod` file has a `replace` directive that points to this local fork for these critical features.

## Step 1: Clone and Build

```bash
# Clone the repository
cd ~/go/src/github.com/relabs-tech
git clone https://github.com/relabs-tech/inertial_computer.git
cd inertial_computer

# Build all components
go build -o imu_producer ./cmd/imu_producer/
go build -o gps_producer ./cmd/gps_producer/
go build -o web ./cmd/web/
go build -o calibration ./cmd/calibration/
go build -o display ./cmd/display/
go build -o register_debug ./cmd/register_debug/
```

## Step 2: Configure

Edit `inertial_config.txt` with your settings:

```bash
nano inertial_config.txt
```

Key settings to verify:
- `MQTT_BROKER` - MQTT broker address (default: `tcp://localhost:1883`)
- `WEB_SERVER_PORT` - Web UI port (default: `8080`)
- `GPS_SERIAL_DEVICE` - GPS serial port (e.g., `/dev/ttyAMA0`)
- IMU SPI settings for left and right sensors

## Step 3: Start MQTT Broker

If not already running:

```bash
# Install Mosquitto (first time only)
sudo apt-get update
sudo apt-get install mosquitto mosquitto-clients

# Start Mosquitto
sudo systemctl start mosquitto
sudo systemctl enable mosquitto  # Auto-start on boot

# Verify it's running
sudo systemctl status mosquitto
```

## Step 4: Start the System

Open multiple terminal sessions (use `tmux` or `screen` recommended):

### Terminal 1: IMU Producer

```bash
cd ~/go/src/github.com/relabs-tech/inertial_computer
sudo ./imu_producer
```

Expected output:
```
starting IMU producer
Left IMU initialized successfully
Right IMU initialized successfully
Publishing to: imu/left, imu/right
```

### Terminal 2: GPS Producer

```bash
cd ~/go/src/github.com/relabs-tech/inertial_computer
sudo ./gps_producer
```

Expected output:
```
starting GPS producer
GPS initialized successfully
Publishing to: gps/fix
```

### Terminal 3: Web Server

```bash
cd ~/go/src/github.com/relabs-tech/inertial_computer
./web
```

Expected output:
```
starting inertial-computer web server
IMU manager initialized successfully
web: connected to MQTT broker at tcp://localhost:1883
web: subscribed to MQTT topic orientation/left
web: subscribed to MQTT topic orientation/right
...
web: listening on :8080
```

### Terminal 4: Display Consumer (Optional)

If you have SSD1306 OLED displays connected via I2C:

```bash
cd ~/go/src/github.com/relabs-tech/inertial_computer
sudo ./display
```

Expected output:
```
starting inertial-computer display subscriber
display: left display initialized at 0x3C
display: right display initialized at 0x3D
display: connected to MQTT broker at tcp://localhost:1883
display: subscribed to orientation/left
display: subscribed to gps/position
display: starting update loop
```

**Note**: The display consumer requires hardware (SSD1306 displays) and runs independently of the web UI.

### Configuring Display Content

You can configure what data appears on each display by editing `inertial_config.txt`:

```bash
# Display content options: imu_raw_left, imu_raw_right, orientation_left, orientation_right, gps
DISPLAY_LEFT_CONTENT=imu_raw_left
DISPLAY_RIGHT_CONTENT=imu_raw_right
```

**Available content types:**
- `imu_raw_left` - Left IMU raw data (Accel X/Y/Z, Gyro X/Y/Z)
- `imu_raw_right` - Right IMU raw data (Accel X/Y/Z, Gyro X/Y/Z)
- `orientation_left` - Left orientation (Roll, Pitch, Yaw in degrees)
- `orientation_right` - Right orientation (Roll, Pitch, Yaw in degrees)
- `gps` - GPS position (Latitude, Longitude, Altitude)

**Default configuration:** Raw left IMU on left display, raw right IMU on right display.

## Step 5: Access the Dashboard

Open a web browser and navigate to:

```
http://localhost:8080
```

Or from another device on the same network:

```
http://<raspberry-pi-ip>:8080
```

You should see:
- Real-time IMU sensor readings
- GPS position and satellite data
- Environmental sensor data (BMP280)
- 3D orientation visualization

## Step 6: Calibrate IMUs (Recommended)

For best accuracy, calibrate your IMU sensors:

1. Click the **"🎯 Calibrate IMU"** button in the top-right corner
2. Select which IMU to calibrate (Left or Right)
3. Follow the on-screen 3D-guided instructions:
   - **Gyroscope**: Static placement + 3 axis rotations
   - **Accelerometer**: 6-point orientation holds
   - **Magnetometer**: Figure-8 motion for 20 seconds
4. Calibration results saved to `{imu}_{timestamp}_inertial_calibration.json`

See [CALIBRATION_UI.md](CALIBRATION_UI.md) for detailed calibration instructions.

## Step 7: Debug IMU Registers (Advanced)

For hardware debugging and configuration experimentation, use the register debug tool:

### What is Register Debugging?

The register debug tool provides direct access to all 128 MPU9250 registers with:
- Real-time read/write operations
- Bitfield manipulation with toggle switches
- Live sensor data monitoring
- SPI speed control
- Configuration export/import

### Starting the Register Debugger

```bash
# Build the register debug tool
go build -o register_debug ./cmd/register_debug/

# Run with sudo (required for SPI hardware access)
sudo ./register_debug

# Access in browser
http://localhost:8081
```

**Note**: Can run concurrently with `imu_producer` - hardware access is thread-safe via IMUManager mutex.

### Features

#### Register Table
- **Read All Registers**: Bulk read all 128 registers (0x00-0x7F)
- **Individual Reads**: Click on any register to read current value
- **Write Protection**: Read-only registers clearly marked
- **Multiple Formats**: View values in hex, binary, and decimal

#### Bitfield Manipulation
For configuration registers (CONFIG, GYRO_CONFIG, ACCEL_CONFIG, etc.):
- Expand register rows to see individual bitfields
- Toggle switches for each bitfield
- Real-time value computation and preview
- Apply button writes computed value to hardware
- Detailed descriptions of each bitfield function

#### SPI Speed Control
Adjust SPI bus speeds for debugging timing issues:
- **Read Speed**: Speed for register reads (default: 1 MHz)
- **Write Speed**: Speed for register writes (default: 500 kHz)
- **Presets**: Fast (4MHz/1MHz), Normal (1MHz/500kHz), Slow (500kHz/250kHz)
- Useful for troubleshooting communication problems

#### Live Sensor Monitoring
Watch real-time sensor data as you modify registers:
- **Accelerometer**: X, Y, Z with current range setting
- **Gyroscope**: X, Y, Z with current range setting
- **Magnetometer**: X, Y, Z with field magnitude
- Updates at 100ms intervals

#### Configuration Management
- **Export**: Save all register values as timestamped JSON file
- **Factory Reset**: Restore default values for all registers
- **Quick Presets**: Apply common configurations (high-precision, low-power, etc.)

### Common Use Cases

#### Experiment with Sensor Ranges
Toggle `ACCEL_FS_SEL` bitfield in ACCEL_CONFIG register (0x1C):
- 0 = ±2g
- 1 = ±4g
- 2 = ±8g (default)
- 3 = ±16g

Toggle `GYRO_FS_SEL` bitfield in GYRO_CONFIG register (0x1B):
- 0 = ±250°/s
- 1 = ±500°/s (default)
- 2 = ±1000°/s
- 3 = ±2000°/s

#### Adjust Digital Low-Pass Filter
Modify `DLPF_CFG` bitfield in CONFIG register (0x1A):
- 0 = 250 Hz
- 1 = 184 Hz
- 2 = 92 Hz
- 3 = 41 Hz (default)
- 4 = 20 Hz
- 5 = 10 Hz
- 6 = 5 Hz

#### Debug I2C Master Settings
For magnetometer communication, tune I2C_MST_CTRL register (0x24):
- Adjust `I2C_MST_CLK` bitfield to change I2C clock speed
- Modify `WAIT_FOR_ES` to control timing behavior

#### Recover from IMU Lockup
If IMU stops responding:
1. Click **"Reinitialize IMU"** button
2. Resets hardware without restarting application
3. Reapplies default configuration

### Safety Features

- **Read-only indicators**: Cannot write to R registers
- **Bitfield validation**: Ensures valid bit patterns
- **Confirmation dialogs**: For critical registers (PWR_MGMT, INT_PIN_CFG)
- **Current value display**: Shows state before modification
- **Revert capability**: Reload all registers to see hardware state
- **Export/restore**: Save and restore working configurations

### Integration with Main Dashboard

Access register debugger from main dashboard at `http://localhost:8080`:
- Click **"Debug Registers"** button in header
- Opens register debug tool in new tab at `http://localhost:8081`

### Navigation from Command Line

```bash
# Link from main dashboard (already implemented in web/index.html)
# Or access directly
firefox http://localhost:8081  # or your browser of choice
```

## Using tmux (Recommended for Multiple Processes)

Instead of multiple terminal windows, use `tmux`:

```bash
# Install tmux (first time only)
sudo apt-get install tmux

# Start a new tmux session
tmux new -s inertial

# Terminal 1: Start IMU producer
sudo ./imu_producer

# Split window horizontally (Ctrl+B then ")
# Terminal 2: Start GPS producer
sudo ./gps_producer

# Split window horizontally again (Ctrl+B then ")
# Terminal 3: Start web server
./web

# Optional: Split window for display consumer (Ctrl+B then ")
# Terminal 4: Start display consumer (if hardware available)
sudo ./display

# Navigate between panes: Ctrl+B then arrow keys
# Detach from session: Ctrl+B then D
# Reattach later: tmux attach -t inertial
```

## Troubleshooting

### IMU Producer Fails to Start

```bash
# Check SPI is enabled
ls /dev/spidev*
# Should show: /dev/spidev0.0  /dev/spidev0.1

# Enable SPI if missing
sudo raspi-config
# Navigate to: Interface Options > SPI > Enable
```

### GPS Producer Fails

```bash
# Check GPS device exists
ls -l /dev/ttyAMA0  # or your configured device

# Check permissions
sudo usermod -a -G dialout $USER
# Log out and back in for group changes to take effect

# Test GPS manually
cat /dev/ttyAMA0
# Should see NMEA sentences like: $GPGGA,123519,4807.038,N,...
```

### MQTT Connection Fails

```bash
# Check MQTT broker is running
sudo systemctl status mosquitto

# Test MQTT manually
mosquitto_sub -h localhost -t '#' -v
# Open another terminal and publish:
mosquitto_pub -h localhost -t test -m "hello"
# Should see: test hello
```

### Web Dashboard Shows No Data

```bash
# Check all producers are running and publishing
mosquitto_sub -h localhost -t 'imu/#' -v
mosquitto_sub -h localhost -t 'gps/#' -v

# Check web server is subscribed
# Look for "subscribed to MQTT topic" messages in web server output
```

### Permission Denied Errors

```bash
# Run IMU and GPS producers with sudo (required for hardware access)
sudo ./imu_producer
sudo ./gps_producer

# Web server doesn't need sudo
./web
```

## Systemd Services (Auto-Start on Boot)

Create systemd service files for automatic startup:

### `/etc/systemd/system/inertial-imu.service`

```ini
[Unit]
Description=Inertial Computer IMU Producer
After=network.target mosquitto.service

[Service]
Type=simple
User=root
WorkingDirectory=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer
ExecStart=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer/imu_producer
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### `/etc/systemd/system/inertial-gps.service`

```ini
[Unit]
Description=Inertial Computer GPS Producer
After=network.target mosquitto.service

[Service]
Type=simple
User=root
WorkingDirectory=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer
ExecStart=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer/gps_producer
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### `/etc/systemd/system/inertial-web.service`

```ini
[Unit]
Description=Inertial Computer Web Server
After=network.target mosquitto.service

[Service]
Type=simple
User=dalarub
WorkingDirectory=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer
ExecStart=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer/web
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Optional: Display Consumer Service** (`/etc/systemd/system/inertial-display.service`)

```ini
[Unit]
Description=Inertial Computer Display Consumer
After=network.target mosquitto.service
Requires=mosquitto.service

[Service]
Type=simple
User=root
WorkingDirectory=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer
ExecStart=/home/dalarub/go/src/github.com/relabs-tech/inertial_computer/display
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**Note**: Display service runs as root for I2C hardware access.

### Enable and Start Services

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services (auto-start on boot)
sudo systemctl enable inertial-imu
sudo systemctl enable inertial-gps
sudo systemctl enable inertial-web
sudo systemctl enable inertial-display  # Optional

# Start services now
sudo systemctl start inertial-imu
sudo systemctl start inertial-gps
sudo systemctl start inertial-web
sudo systemctl start inertial-display  # Optional

# Check status
sudo systemctl status inertial-imu
sudo systemctl status inertial-gps
sudo systemctl status inertial-web

# View logs
sudo journalctl -u inertial-imu -f
sudo journalctl -u inertial-gps -f
sudo journalctl -u inertial-web -f
```

## Performance Tuning

### Increase IMU Sample Rate

Edit `cmd/imu_producer/main.go`:
```go
ticker := time.NewTicker(10 * time.Millisecond)  // 100 Hz instead of 100ms (10 Hz)
```

### Reduce Web Dashboard Update Rate

Edit `web/index.html`:
```javascript
setInterval(tick, 100);  // 100ms instead of 500ms
```

### MQTT QoS Settings

For real-time performance, QoS 0 (fire and forget) is recommended and already configured.

For reliability, change to QoS 1 in producer code:
```go
token := client.Publish(topic, 1, false, payload)  // QoS 1
```

## Next Steps

- **Calibrate IMUs**: Use the web UI calibration tool for accurate readings
- **Configure Fusion**: Edit orientation fusion parameters in config
- **View Logs**: Monitor MQTT topics with `mosquitto_sub -t '#' -v`
- **Add Sensors**: Extend with additional BMP280 sensors or other I2C/SPI devices
- **Remote Access**: Set up port forwarding or VPN for remote dashboard access

## Useful Commands

```bash
# View all MQTT traffic
mosquitto_sub -h localhost -t '#' -v

# Monitor specific topics
mosquitto_sub -h localhost -t 'imu/left' -v
mosquitto_sub -h localhost -t 'gps/fix' -v

# Check process status
ps aux | grep imu_producer
ps aux | grep gps_producer
ps aux | grep web

# Kill all processes
pkill -f imu_producer
pkill -f gps_producer
pkill -f web

# Rebuild everything
go build -o imu_producer ./cmd/imu_producer/
go build -o gps_producer ./cmd/gps_producer/
go build -o web ./cmd/web/

# Clean build
go clean
go build ./...
```

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/relabs-tech/inertial_computer/issues
- Documentation: See README.md and ARCHITECTURE.md

## Quick Reference Card

| Component | Command | Port/Topic | Notes |
|-----------|---------|------------|-------|
| MQTT Broker | `mosquitto` | 1883 | Must start first |
| IMU Producer | `sudo ./imu_producer` | → `imu/left`, `imu/right` | Needs sudo |
| GPS Producer | `sudo ./gps_producer` | → `gps/fix` | Needs sudo |
| Web Server | `./web` | :8080 | No sudo needed |
| Dashboard | Browser | http://localhost:8080 | - |
| Calibration UI | Browser | http://localhost:8080/calibration.html | - |
| Calibration CLI | `sudo ./calibration` | - | Alternative to web UI |
| Register Debug | `sudo ./register_debug` | :8081 | Hardware debugging |
| Register Debug UI | Browser | http://localhost:8081 | Direct register access |
