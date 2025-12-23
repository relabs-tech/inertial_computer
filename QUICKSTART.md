# Quick Start Guide

This guide will get your Inertial Computer system up and running in minutes.

## Prerequisites

- Go 1.21 or later installed
- Raspberry Pi with MPU9250 IMUs connected via SPI
- GPS module connected via serial
- MQTT broker running (Mosquitto recommended)

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

### Enable and Start Services

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable services (auto-start on boot)
sudo systemctl enable inertial-imu
sudo systemctl enable inertial-gps
sudo systemctl enable inertial-web

# Start services now
sudo systemctl start inertial-imu
sudo systemctl start inertial-gps
sudo systemctl start inertial-web

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
