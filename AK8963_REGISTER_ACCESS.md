# AK8963 Magnetometer Register Access Implementation

## Overview
Successfully implemented complete low-level register access for the AK8963 magnetometer via I2C master protocol on the MPU9250. The magnetometer is now accessible for direct register reads/writes through both the register debug WebUI and programmatic API.

## Architecture

### I2C Master Configuration
The MPU9250 acts as an I2C master to communicate with the AK8963 slave via the following protocol:

**Register Read Sequence:**
1. Write AK8963 address with read bit (0x0C | 0x80) to I2C_SLV0_ADDR (0x25)
2. Write target AK8963 register address to I2C_SLV0_REG (0x26)
3. Write control byte 0x81 (enable, 1 byte) to I2C_SLV0_CTRL (0x27)
4. Wait 2ms for transaction completion
5. Read result from EXT_SENS_DATA_00 (0x49)

**Register Write Sequence:**
1. Write AK8963 address without read bit (0x0C) to I2C_SLV0_ADDR (0x25)
2. Write target AK8963 register address to I2C_SLV0_REG (0x26)
3. Write data byte to I2C_SLV0_DO (0x28)
4. Write control byte 0x81 to I2C_SLV0_CTRL (0x27)
5. Wait 2ms for transaction completion

### Code Organization

**File: `internal/sensors/imu_source.go`** (imuSource implementation)
- `ReadAK8963Register(regAddr byte) (byte, error)` — Single register read
- `WriteAK8963Register(regAddr, value byte) error` — Single register write
- `ReadAllAK8963Registers() (map[byte]byte, error)` — Bulk read of all 16 registers
- Uses I2C master configuration with proper timing (2ms sleep)

**File: `internal/sensors/imu.go`** (IMUManager delegation)
- `ReadAK8963Register(imuID string, regAddr byte) (byte, error)`
- `WriteAK8963Register(imuID string, regAddr, value byte) error`
- `ReadAllAK8963Registers(imuID string) (map[byte]byte, error)`
- Thread-safe delegating methods with RWMutex protection

**File: `internal/app/register_debug_handler.go`** (WebSocket handler routing)
- `handleRead()` — Routes device parameter to ReadAK8963Register or ReadRegister
- `handleReadAll()` — Routes device parameter to ReadAllAK8963Registers or ReadAllRegisters
- `handleWrite()` — Routes device parameter to WriteAK8963Register or WriteRegister
- Device parameter parsing: `device == "ak8963"` for magnetometer, default "mpu9250" for IMU

**File: `web/register_debug.html`** (WebUI device selector)
- Device selector dropdown switching between "mpu9250" and "ak8963"
- Global `currentDevice` variable tracks selection
- All WebSocket messages include `device: currentDevice` parameter
- Register map switches dynamically based on device selection

**File: `internal/sensors/mpu9250_registers.go`** (Register metadata)
- `GetAK8963RegisterMap()` — Returns 14 AK8963 register definitions with bitfield metadata
- AK8963 registers: 0x00-0x01 (ID/INFO), 0x02-0x09 (STATUS/DATA), 0x0A-0x0C (CONTROL), 0x10-0x12 (CALIBRATION)

## AK8963 Register Map

| Address | Name | Access | Default | Description |
|---------|------|--------|---------|-------------|
| 0x00 | WIA | R | 0x48 | WHO_AM_I - Sensor ID |
| 0x01 | INFO | R | - | Information register |
| 0x02 | ST1 | R | - | Status 1 (data ready, overflow) |
| 0x03-0x08 | H*L/H*H | R | - | Magnetometer X/Y/Z LSB/MSB |
| 0x09 | ST2 | R | - | Status 2 (overflow, self-test) |
| 0x0A | CNTL1 | RW | 0x00 | Control 1 (mode, resolution) |
| 0x0B | CNTL2 | RW | 0x01 | Control 2 (reset) |
| 0x0C | ASTC | RW | 0x00 | Self-test control |
| 0x10-0x12 | ASAX/Y/Z | R | - | Sensitivity adjustment (calibration) |

## WebSocket Protocol

### Device Selection
```json
{"action": "get_map", "device": "ak8963"}
→ Returns: AK8963 register metadata (14 registers)

{"action": "get_map", "device": "mpu9250"}
→ Returns: MPU9250 register metadata (128 registers)
```

### Register Operations
```json
// Read AK8963 register
{"action": "read", "device": "ak8963", "imu": "left", "addr": "0x00"}
→ Returns: {"type": "register_data", "addr": "0x00", "value": "0x48", ...}

// Read all AK8963 registers
{"action": "read_all", "device": "ak8963", "imu": "left"}
→ Returns: {"type": "register_data", "registers": {"0x00": "0x48", ...}}

// Write AK8963 register
{"action": "write", "device": "ak8963", "imu": "left", "addr": "0x0A", "value": "0x01"}
→ Returns: Confirmation response
```

## Data Flow

```
WebUI Device Selector
    ↓
WebSocket Message (device="ak8963")
    ↓
register_debug_handler.go (parseDeviceParameter)
    ↓
if device == "ak8963":
    IMUManager.ReadAK8963Register()
else:
    IMUManager.ReadRegister()
    ↓
imuSource.ReadAK8963Register()
    ↓
I2C Master Configuration Sequence
    ↓
Read Result from EXT_SENS_DATA_00
```

## Testing

### Manual Testing via WebUI
1. Open http://localhost:8081
2. Select "AK8963" from device dropdown
3. Click "Read All Registers"
4. Verify response contains 14 registers with values
5. Expected WHO_AM_I (0x00): 0x48 for both left and right sensors

### Programmatic Testing
```go
mgr := sensors.GetIMUManager()
mgr.Init()

// Read single register
whoami, _ := mgr.ReadAK8963Register("left", 0x00)
fmt.Printf("WHO_AM_I: 0x%02X\n", whoami)

// Read all registers
regs, _ := mgr.ReadAllAK8963Registers("left")
for addr, val := range regs {
    fmt.Printf("0x%02X: 0x%02X\n", addr, val)
}

// Write register
mgr.WriteAK8963Register("left", 0x0A, 0x01)
```

## Key Features

✅ **Device Isolation** — MPU9250 and AK8963 registers accessed via separate code paths
✅ **Thread Safety** — IMUManager uses RWMutex for concurrent access
✅ **Error Handling** — Comprehensive error messages for I2C transaction failures
✅ **Timing Correctness** — 2ms sleep between I2C slave setup and result read
✅ **WebUI Integration** — Dropdown device selector with dynamic register map switching
✅ **Bulk Operations** — ReadAllAK8963Registers() for efficient multi-register reads
✅ **Backward Compatibility** — Default device="mpu9250" maintains existing behavior

## Known Limitations

- AK8963 write operations execute immediately (no confirmation check)
- Single-byte reads/writes only (standard I2C slave protocol)
- No automatic overflow handling (ST1/ST2 overflow bits must be checked manually)
- I2C master timeout not implemented (2ms fixed wait)
- No DMA support (single byte at a time)

## Future Enhancements

1. **Multi-byte reads** — Bulk read magnetometer data (0x03-0x08) in single I2C transaction
2. **Overflow detection** — Automatic ST1 ready flag polling
3. **Continuous mode support** — Configure sensor for continuous measurement (0x0A CNTL1)
4. **Sensitivity adjustment** — Apply ASAX/ASAY/ASAZ calibration values to raw measurements
5. **Temperature compensation** — Account for temperature drift (if temp sensor available)
6. **Fusion integration** — Use calibrated magnetometer data in yaw calculation

## Files Modified

- ✅ `internal/sensors/imu_source.go` — Added 3 AK8963 methods (ReadAK8963Register, WriteAK8963Register, ReadAllAK8963Registers)
- ✅ `internal/sensors/imu.go` — Added 3 delegation methods with thread safety
- ✅ `internal/app/register_debug_handler.go` — Updated `handleRead()`, `handleReadAll()`, `handleWrite()`, `get_map` routing
- ✅ `web/register_debug.html` — Added device selector, device parameter passing
- ✅ `internal/sensors/mpu9250_registers.go` — Added AK8963 register map

## Commits

1. "Add register access helpers for debugger" (devices repo)
2. "Move register access methods to imuSource" (inertial_computer repo)
3. "Add AK8963 register write support and device parameter routing in WebSocket handler" (inertial_computer repo)

