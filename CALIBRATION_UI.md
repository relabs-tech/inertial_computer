# Web-Based IMU Calibration

The Inertial Computer now includes an interactive web-based calibration interface for IMU sensors.

## Features

- **3D Visualization**: Real-time 3D device orientation display using Three.js
- **Guided Workflow**: Step-by-step instructions for gyroscope, accelerometer, and magnetometer calibration
- **Progress Tracking**: Visual progress bars and confidence metrics
- **Real-Time Updates**: WebSocket-based communication for instant feedback
- **Animated Instructions**: 3D device animation shows exact orientation required for each step

## Access

1. Start the web server: `./web_server`
2. Open browser to `http://localhost:8080` (or configured port)
3. Click the "🎯 Calibrate IMU" button in the top-right corner
4. Select which IMU to calibrate (Left or Right)

## Calibration Process

### Phase 1: Gyroscope Calibration
1. **Static**: Place device on flat surface, keep still for 10 seconds
2. **X-Axis Rotation**: Slowly rotate around X-axis (pitch)
3. **Y-Axis Rotation**: Slowly rotate around Y-axis (roll)
4. **Z-Axis Rotation**: Slowly rotate around Z-axis (yaw)

### Phase 2: Accelerometer Calibration
Six-point calibration holding device in each orientation for 5 seconds:
1. **Z+ Up**: Flat with Z-axis pointing up
2. **Z- Down**: Upside down with Z-axis pointing down
3. **X+ Right**: Rotated with X-axis pointing up
4. **X- Left**: Rotated with X-axis pointing down
5. **Y+ Forward**: Rotated with Y-axis pointing up
6. **Y- Back**: Rotated with Y-axis pointing down

### Phase 3: Magnetometer Calibration
1. **Figure-8 Motion**: Move device in figure-8 pattern for 20 seconds
2. Rotate through as many orientations as possible
3. Covers all three axes for hard-iron and soft-iron calibration

## Output

Calibration results are saved to: `{imu}_{timestamp}_inertial_calibration.json`

Example: `left_1703345678_inertial_calibration.json`

The JSON file contains:
- Gyroscope bias corrections (X, Y, Z)
- Accelerometer bias and scale factors (X, Y, Z)
- Magnetometer offset and scale factors (X, Y, Z)
- Confidence metrics for each sensor
- Statistical data (standard deviations, sample counts)

## Architecture

### Backend
- **WebSocket Handler**: `/api/calibration/ws` ([internal/app/calibration_handler.go](../internal/app/calibration_handler.go))
- **State Machine**: Manages calibration phases and steps
- **IMU Integration**: Uses `sensors.IMUManager` for direct sensor access
- **Real-Time Sampling**: 100ms intervals, configurable sample counts

### Frontend
- **HTML/CSS**: [web/calibration.html](../web/calibration.html)
- **Three.js**: 3D device visualization
- **WebSocket Client**: Bidirectional communication
- **Responsive Design**: Works on desktop and mobile

### Communication Protocol

WebSocket messages (client → server):
```json
{"action": "init", "imu": "left"}     // Initialize calibration
{"action": "next"}                     // Proceed to next step
{"action": "cancel"}                   // Cancel calibration
```

WebSocket messages (server → client):
```json
{"type": "phase", "phase": "gyro"}                    // Current phase
{"type": "step", "step": "gyro-x", "phase": "gyro"}  // Current step
{"type": "progress", "progress": 45.2}                // Progress %
{"type": "stats", "stats": {...}}                     // Live statistics
{"type": "action", "message": "ready"}                // Enable next button
{"type": "complete", "results": {...}}                // Calibration done
{"type": "error", "message": "..."}                   // Error occurred
```

## Integration with Web Server

The calibration handler is integrated into [cmd/web/main.go](../cmd/web/main.go):
1. Configuration loaded via `config.InitGlobal()`
2. IMU manager initialized via `sensors.GetIMUManager().Init()`
3. WebSocket endpoint registered at `/api/calibration/ws`
4. Static files served from `web/` directory

## Button Integration

The main dashboard ([web/index.html](../web/index.html)) includes a calibration button in the header:
```html
<button class="cal-button" onclick="window.location.href='/calibration.html'">
  🎯 Calibrate IMU
</button>
```

## Development Notes

- Calibration runs on same server as main dashboard
- IMU sensors must be initialized for calibration to work
- WebSocket connection required for real-time updates
- All calculations performed on server side
- Frontend handles only visualization and user interaction

## Future Enhancements

- [ ] Apply calibration directly to sensor readings
- [ ] Calibration profiles management (save/load/switch)
- [ ] Live sensor data visualization during calibration
- [ ] Automatic quality assessment and retry suggestions
- [ ] Multi-device calibration synchronization
- [ ] Export calibration to different formats
