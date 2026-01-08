// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config holds all application configuration values.
type Config struct {
	// MQTT
	MQTTBroker           string
	MQTTClientIDProducer string
	MQTTClientIDGPS      string
	MQTTClientIDConsole  string
	MQTTClientIDWeb      string
	MQTTClientIDDisplay  string
	MQTTClientIDHMC      string

	// Topics
	TopicPoseLeft          string
	TopicPoseRight         string
	TopicPoseFused         string
	TopicIMULeft           string
	TopicIMURight          string
	TopicMagLeft           string
	TopicMagRight          string
	TopicBMPLeft           string
	TopicBMPRight          string
	TopicGPSPosition       string
	TopicGPSVelocity       string
	TopicGPSQuality        string
	TopicGPSSatellites     string
	TopicGLONASSSatellites string
	TopicGPS               string
	// External magnetometer topic
	TopicMagHMC            string

	// HMC5983 external magnetometer
	HMCI2CBus         int
	HMCI2CAddr        uint16
	HMCODRHz          int
	HMCAvgSamples     int
	HMCGainCode       int
	HMCMode           string
	HMCSampleInterval int // milliseconds

	// IMU Hardware
	IMULeftSPIDevice  string
	IMULeftCSPin      string
	IMURightSPIDevice string
	IMURightCSPin     string

	// IMU Sensor Ranges
	// Accelerometer: 0=±2g, 1=±4g, 2=±8g, 3=±16g
	IMUAccelRange byte
	// Gyroscope: 0=±250°/s, 1=±500°/s, 2=±1000°/s, 3=±2000°/s
	IMUGyroRange byte

	// IMU Sample Rate Configuration
	IMUDLPFConfig    byte // Digital Low Pass Filter configuration (0-7)
	IMUSampleRateDiv byte // Sample rate divider (output rate = internal rate / (1 + div))
	IMUAccelDLPF     byte // Accelerometer DLPF configuration (0-7)

	// BMP Hardware
	BMPLeftSPIDevice  string
	BMPRightSPIDevice string

	// BMP Left Configuration
	BMPLeftPressureOSR byte
	BMPLeftTempOSR     byte
	BMPLeftMode        byte
	BMPLeftIIRFilter   byte
	BMPLeftStandbyTime byte

	// BMP Right Configuration
	BMPRightPressureOSR byte
	BMPRightTempOSR     byte
	BMPRightMode        byte
	BMPRightIIRFilter   byte
	BMPRightStandbyTime byte

	// GPS
	GPSSerialPort string
	GPSBaudRate   int

	// Magnetometer Configuration
	MagWriteDelayMS      int  // Delay after magnetometer write operations (ms)
	MagReadDelayMS       int  // Delay for I2C master read completion (ms)
	MagScale             byte // Resolution: 0=14-bit, 1=16-bit
	MagMode              byte // Operating mode: 0x02=8Hz, 0x06=100Hz continuous
	MagSampleRateDivider byte // I2C master read frequency divider (0-15)

	// Register Debug Overrides
	RegisterDebugMagWriteDelay int  // Experimental write delay override (-1 = use MAG_WRITE_DELAY_MS)
	RegisterDebugMagReadDelay  int  // Experimental read delay override (-1 = use MAG_READ_DELAY_MS)
	RegisterDebugMagUnsafeMode bool // Allow unsafe magnetometer operations in register debug

	// Timing
	IMUSampleInterval  int // milliseconds
	ConsoleLogInterval int // milliseconds

	// Web Server
	WebServerPort                int
	WeatherUpdateIntervalMinutes int

	// Display
	DisplayLeftI2CAddr    uint16
	DisplayRightI2CAddr   uint16
	DisplayUpdateInterval int    // milliseconds
	DisplayLeftContent    string // what to show: "imu_raw_left", "imu_raw_right", "orientation_left", "orientation_right", "gps"
	DisplayRightContent   string // what to show: "imu_raw_left", "imu_raw_right", "orientation_left", "orientation_right", "gps"

	// Register Debugging Topics
	TopicRegistersCmdRead     string
	TopicRegistersCmdWrite    string
	TopicRegistersCmdInit     string
	TopicRegistersCmdSPISpeed string
	TopicRegistersDataLeft    string
	TopicRegistersDataRight   string
	TopicRegistersMap         string
	TopicRegistersStatus      string

	// Register Debugging Configuration
	RegisterDebugAllowedRanges     string // e.g., "0x1B-0x1D,0x6B" - writable register ranges
	RegisterDebugDefaultReadSpeed  int64  // Hz
	RegisterDebugDefaultWriteSpeed int64  // Hz
	RegisterDebugMaxSPISpeed       int64  // Hz
	RegisterDebugMinSPISpeed       int64  // Hz
	IMULeftRegisterConfigFile      string // path to register config JSON file
	IMURightRegisterConfigFile     string // path to register config JSON file
}

// Package-level unexported variables for singleton pattern:
//   - globalConfig: unexported (lowercase) so other packages cannot access it directly.
//     This enforces encapsulation and prevents external code from modifying config without proper locking.
//     Has package-level scope (visible to all functions in this package, persists for program lifetime).
//   - configOnce: ensures InitGlobal() only runs once, even if called multiple times.
//   - configMu: RWMutex protects concurrent access. Write lock (Lock) for initialization,
//     read lock (RLock) for Get() allows multiple concurrent readers without blocking each other.
//
// External code must use InitGlobal() to set and Get() to read, ensuring thread safety.
var (
	globalConfig *Config
	configOnce   sync.Once
	configMu     sync.RWMutex
)

// Load reads the configuration file and returns a Config struct.
func Load(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config line %d: %q", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if err := cfg.setValue(key, value); err != nil {
			return nil, fmt.Errorf("config line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setValue sets a config value based on the key.
func (c *Config) setValue(key, value string) error {
	switch key {
	// MQTT
	case "MQTT_BROKER":
		c.MQTTBroker = value
	case "MQTT_CLIENT_ID_PRODUCER":
		c.MQTTClientIDProducer = value
	case "MQTT_CLIENT_ID_GPS":
		c.MQTTClientIDGPS = value
	case "MQTT_CLIENT_ID_CONSOLE":
		c.MQTTClientIDConsole = value
	case "MQTT_CLIENT_ID_WEB":
		c.MQTTClientIDWeb = value
	case "MQTT_CLIENT_ID_DISPLAY":
		c.MQTTClientIDDisplay = value
	case "MQTT_CLIENT_ID_HMC":
		c.MQTTClientIDHMC = value

	// Topics
	case "TOPIC_POSE_LEFT":
		c.TopicPoseLeft = value
	case "TOPIC_POSE_RIGHT":
		c.TopicPoseRight = value
	case "TOPIC_POSE_FUSED":
		c.TopicPoseFused = value
	case "TOPIC_IMU_LEFT":
		c.TopicIMULeft = value
	case "TOPIC_IMU_RIGHT":
		c.TopicIMURight = value
	case "TOPIC_MAG_LEFT":
		c.TopicMagLeft = value
	case "TOPIC_MAG_RIGHT":
		c.TopicMagRight = value
	case "TOPIC_BMP_LEFT":
		c.TopicBMPLeft = value
	case "TOPIC_BMP_RIGHT":
		c.TopicBMPRight = value
	case "TOPIC_GPS_POSITION":
		c.TopicGPSPosition = value
	case "TOPIC_GPS_VELOCITY":
		c.TopicGPSVelocity = value
	case "TOPIC_GPS_QUALITY":
		c.TopicGPSQuality = value
	case "TOPIC_GPS_SATELLITES":
		c.TopicGPSSatellites = value
	case "TOPIC_GLONASS_SATELLITES":
		c.TopicGLONASSSatellites = value
	case "TOPIC_GPS":
		c.TopicGPS = value
	case "TOPIC_MAG_HMC":
		c.TopicMagHMC = value

	// HMC5983 external magnetometer
	case "HMC_I2C_BUS":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HMC_I2C_BUS %q: %w", value, err)
		}
		c.HMCI2CBus = v
	case "HMC_I2C_ADDR":
		addr, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return fmt.Errorf("invalid HMC_I2C_ADDR %q: %w", value, err)
		}
		c.HMCI2CAddr = uint16(addr)
	case "HMC_ODR_HZ":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HMC_ODR_HZ %q: %w", value, err)
		}
		c.HMCODRHz = v
	case "HMC_AVG_SAMPLES":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HMC_AVG_SAMPLES %q: %w", value, err)
		}
		c.HMCAvgSamples = v
	case "HMC_GAIN_CODE":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HMC_GAIN_CODE %q: %w", value, err)
		}
		c.HMCGainCode = v
	case "HMC_MODE":
		c.HMCMode = value
	case "HMC_SAMPLE_INTERVAL":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid HMC_SAMPLE_INTERVAL %q: %w", value, err)
		}
		c.HMCSampleInterval = v

	// IMU Hardware
	case "IMU_LEFT_SPI_DEVICE":
		c.IMULeftSPIDevice = value
	case "IMU_LEFT_CS_PIN":
		c.IMULeftCSPin = value
	case "IMU_RIGHT_SPI_DEVICE":
		c.IMURightSPIDevice = value
	case "IMU_RIGHT_CS_PIN":
		c.IMURightCSPin = value

	// IMU Sensor Ranges
	case "IMU_ACCEL_RANGE":
		rangeVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_ACCEL_RANGE %q: %w", value, err)
		}
		if rangeVal < 0 || rangeVal > 3 {
			return fmt.Errorf("IMU_ACCEL_RANGE must be 0-3 (0=±2g, 1=±4g, 2=±8g, 3=±16g), got %d", rangeVal)
		}
		c.IMUAccelRange = byte(rangeVal)
	case "IMU_GYRO_RANGE":
		rangeVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_GYRO_RANGE %q: %w", value, err)
		}
		if rangeVal < 0 || rangeVal > 3 {
			return fmt.Errorf("IMU_GYRO_RANGE must be 0-3 (0=±250°/s, 1=±500°/s, 2=±1000°/s, 3=±2000°/s), got %d", rangeVal)
		}
		c.IMUGyroRange = byte(rangeVal)

	// IMU Sample Rate Configuration
	case "IMU_DLPF_CFG":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_DLPF_CFG %q: %w", value, err)
		}
		if val < 0 || val > 7 {
			return fmt.Errorf("IMU_DLPF_CFG must be 0-7, got %d", val)
		}
		c.IMUDLPFConfig = byte(val)
	case "IMU_SMPLRT_DIV":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_SMPLRT_DIV %q: %w", value, err)
		}
		if val < 0 || val > 255 {
			return fmt.Errorf("IMU_SMPLRT_DIV must be 0-255, got %d", val)
		}
		c.IMUSampleRateDiv = byte(val)
	case "IMU_ACCEL_DLPF":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_ACCEL_DLPF %q: %w", value, err)
		}
		if val < 0 || val > 7 {
			return fmt.Errorf("IMU_ACCEL_DLPF must be 0-7, got %d", val)
		}
		c.IMUAccelDLPF = byte(val)

	// BMP Hardware
	case "BMP_LEFT_SPI_DEVICE":
		c.BMPLeftSPIDevice = value
	case "BMP_RIGHT_SPI_DEVICE":
		c.BMPRightSPIDevice = value

	// BMP Left Configuration
	case "BMP_LEFT_PRESSURE_OSR":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_LEFT_PRESSURE_OSR %q: %w", value, err)
		}
		if val < 0 || val > 5 {
			return fmt.Errorf("BMP_LEFT_PRESSURE_OSR must be 0-5, got %d", val)
		}
		c.BMPLeftPressureOSR = byte(val)
	case "BMP_LEFT_TEMP_OSR":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_LEFT_TEMP_OSR %q: %w", value, err)
		}
		if val < 0 || val > 5 {
			return fmt.Errorf("BMP_LEFT_TEMP_OSR must be 0-5, got %d", val)
		}
		c.BMPLeftTempOSR = byte(val)
	case "BMP_LEFT_MODE":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_LEFT_MODE %q: %w", value, err)
		}
		if val < 0 || val > 3 {
			return fmt.Errorf("BMP_LEFT_MODE must be 0-3, got %d", val)
		}
		c.BMPLeftMode = byte(val)
	case "BMP_LEFT_IIR_FILTER":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_LEFT_IIR_FILTER %q: %w", value, err)
		}
		if val < 0 || val > 4 {
			return fmt.Errorf("BMP_LEFT_IIR_FILTER must be 0-4, got %d", val)
		}
		c.BMPLeftIIRFilter = byte(val)
	case "BMP_LEFT_STANDBY_TIME":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_LEFT_STANDBY_TIME %q: %w", value, err)
		}
		if val < 0 || val > 7 {
			return fmt.Errorf("BMP_LEFT_STANDBY_TIME must be 0-7, got %d", val)
		}
		c.BMPLeftStandbyTime = byte(val)

	// BMP Right Configuration
	case "BMP_RIGHT_PRESSURE_OSR":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_RIGHT_PRESSURE_OSR %q: %w", value, err)
		}
		if val < 0 || val > 5 {
			return fmt.Errorf("BMP_RIGHT_PRESSURE_OSR must be 0-5, got %d", val)
		}
		c.BMPRightPressureOSR = byte(val)
	case "BMP_RIGHT_TEMP_OSR":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_RIGHT_TEMP_OSR %q: %w", value, err)
		}
		if val < 0 || val > 5 {
			return fmt.Errorf("BMP_RIGHT_TEMP_OSR must be 0-5, got %d", val)
		}
		c.BMPRightTempOSR = byte(val)
	case "BMP_RIGHT_MODE":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_RIGHT_MODE %q: %w", value, err)
		}
		if val < 0 || val > 3 {
			return fmt.Errorf("BMP_RIGHT_MODE must be 0-3, got %d", val)
		}
		c.BMPRightMode = byte(val)
	case "BMP_RIGHT_IIR_FILTER":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_RIGHT_IIR_FILTER %q: %w", value, err)
		}
		if val < 0 || val > 4 {
			return fmt.Errorf("BMP_RIGHT_IIR_FILTER must be 0-4, got %d", val)
		}
		c.BMPRightIIRFilter = byte(val)
	case "BMP_RIGHT_STANDBY_TIME":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid BMP_RIGHT_STANDBY_TIME %q: %w", value, err)
		}
		if val < 0 || val > 7 {
			return fmt.Errorf("BMP_RIGHT_STANDBY_TIME must be 0-7, got %d", val)
		}
		c.BMPRightStandbyTime = byte(val)

	// GPS
	case "GPS_SERIAL_PORT":
		c.GPSSerialPort = value
	case "GPS_BAUD_RATE":
		rate, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid GPS_BAUD_RATE %q: %w", value, err)
		}
		c.GPSBaudRate = rate

	// Magnetometer Configuration
	case "MAG_WRITE_DELAY_MS":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid MAG_WRITE_DELAY_MS %q: %w", value, err)
		}
		if val < 1 || val > 200 {
			return fmt.Errorf("MAG_WRITE_DELAY_MS must be 1-200ms, got %d", val)
		}
		c.MagWriteDelayMS = val
	case "MAG_READ_DELAY_MS":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid MAG_READ_DELAY_MS %q: %w", value, err)
		}
		if val < 1 || val > 200 {
			return fmt.Errorf("MAG_READ_DELAY_MS must be 1-200ms, got %d", val)
		}
		c.MagReadDelayMS = val
	case "MAG_SCALE":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid MAG_SCALE %q: %w", value, err)
		}
		if val < 0 || val > 1 {
			return fmt.Errorf("MAG_SCALE must be 0 or 1, got %d", val)
		}
		c.MagScale = byte(val)
	case "MAG_MODE":
		// Parse hex value (0x06) or decimal
		var val uint64
		var err error
		if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
			val, err = strconv.ParseUint(value[2:], 16, 8)
		} else {
			val, err = strconv.ParseUint(value, 0, 8)
		}
		if err != nil {
			return fmt.Errorf("invalid MAG_MODE %q: %w", value, err)
		}
		// Validate against allowed modes
		validModes := map[byte]bool{
			0x00: true, 0x01: true, 0x02: true,
			0x04: true, 0x06: true, 0x08: true, 0x0F: true,
		}
		if !validModes[byte(val)] {
			return fmt.Errorf("MAG_MODE=0x%02X invalid, must be one of: 0x00, 0x01, 0x02, 0x04, 0x06, 0x08, 0x0F", val)
		}
		c.MagMode = byte(val)
	case "MAG_SAMPLE_RATE_DIVIDER":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid MAG_SAMPLE_RATE_DIVIDER %q: %w", value, err)
		}
		if val < 0 || val > 15 {
			return fmt.Errorf("MAG_SAMPLE_RATE_DIVIDER must be 0-15, got %d", val)
		}
		c.MagSampleRateDivider = byte(val)

	// Register Debug Overrides
	case "REGISTER_DEBUG_MAG_WRITE_DELAY":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_MAG_WRITE_DELAY %q: %w", value, err)
		}
		if val != -1 && (val < 1 || val > 500) {
			return fmt.Errorf("REGISTER_DEBUG_MAG_WRITE_DELAY must be -1 or 1-500ms, got %d", val)
		}
		c.RegisterDebugMagWriteDelay = val
	case "REGISTER_DEBUG_MAG_READ_DELAY":
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_MAG_READ_DELAY %q: %w", value, err)
		}
		if val != -1 && (val < 1 || val > 200) {
			return fmt.Errorf("REGISTER_DEBUG_MAG_READ_DELAY must be -1 or 1-200ms, got %d", val)
		}
		c.RegisterDebugMagReadDelay = val
	case "REGISTER_DEBUG_MAG_UNSAFE_MODE":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_MAG_UNSAFE_MODE %q: %w", value, err)
		}
		c.RegisterDebugMagUnsafeMode = val

	// Timing
	case "IMU_SAMPLE_INTERVAL":
		interval, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid IMU_SAMPLE_INTERVAL %q: %w", value, err)
		}
		c.IMUSampleInterval = interval
	case "CONSOLE_LOG_INTERVAL":
		interval, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid CONSOLE_LOG_INTERVAL %q: %w", value, err)
		}
		c.ConsoleLogInterval = interval

	// Web Server
	case "WEB_SERVER_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid WEB_SERVER_PORT %q: %w", value, err)
		}
		c.WebServerPort = port
	case "WEATHER_UPDATE_INTERVAL_MINUTES":
		minutes, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid WEATHER_UPDATE_INTERVAL_MINUTES %q: %w", value, err)
		}
		c.WeatherUpdateIntervalMinutes = minutes

	// Display
	case "DISPLAY_LEFT_I2C_ADDR":
		addr, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return fmt.Errorf("invalid DISPLAY_LEFT_I2C_ADDR %q: %w", value, err)
		}
		c.DisplayLeftI2CAddr = uint16(addr)
	case "DISPLAY_RIGHT_I2C_ADDR":
		addr, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return fmt.Errorf("invalid DISPLAY_RIGHT_I2C_ADDR %q: %w", value, err)
		}
		c.DisplayRightI2CAddr = uint16(addr)
	case "DISPLAY_UPDATE_INTERVAL":
		interval, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid DISPLAY_UPDATE_INTERVAL %q: %w", value, err)
		}
		c.DisplayUpdateInterval = interval
	case "DISPLAY_LEFT_CONTENT":
		c.DisplayLeftContent = value
	case "DISPLAY_RIGHT_CONTENT":
		c.DisplayRightContent = value

	// Register Debugging Topics
	case "TOPIC_REGISTERS_CMD_READ":
		c.TopicRegistersCmdRead = value
	case "TOPIC_REGISTERS_CMD_WRITE":
		c.TopicRegistersCmdWrite = value
	case "TOPIC_REGISTERS_CMD_INIT":
		c.TopicRegistersCmdInit = value
	case "TOPIC_REGISTERS_CMD_SPI_SPEED":
		c.TopicRegistersCmdSPISpeed = value
	case "TOPIC_REGISTERS_DATA_LEFT":
		c.TopicRegistersDataLeft = value
	case "TOPIC_REGISTERS_DATA_RIGHT":
		c.TopicRegistersDataRight = value
	case "TOPIC_REGISTERS_MAP":
		c.TopicRegistersMap = value
	case "TOPIC_REGISTERS_STATUS":
		c.TopicRegistersStatus = value

	// Register Debugging Configuration
	case "REGISTER_DEBUG_ALLOWED_RANGES":
		c.RegisterDebugAllowedRanges = value
	case "REGISTER_DEBUG_DEFAULT_READ_SPEED":
		speed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_DEFAULT_READ_SPEED %q: %w", value, err)
		}
		c.RegisterDebugDefaultReadSpeed = speed
	case "REGISTER_DEBUG_DEFAULT_WRITE_SPEED":
		speed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_DEFAULT_WRITE_SPEED %q: %w", value, err)
		}
		c.RegisterDebugDefaultWriteSpeed = speed
	case "REGISTER_DEBUG_MAX_SPI_SPEED":
		speed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_MAX_SPI_SPEED %q: %w", value, err)
		}
		c.RegisterDebugMaxSPISpeed = speed
	case "REGISTER_DEBUG_MIN_SPI_SPEED":
		speed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid REGISTER_DEBUG_MIN_SPI_SPEED %q: %w", value, err)
		}
		c.RegisterDebugMinSPISpeed = speed
	case "IMU_LEFT_REGISTER_CONFIG_FILE":
		c.IMULeftRegisterConfigFile = value
	case "IMU_RIGHT_REGISTER_CONFIG_FILE":
		c.IMURightRegisterConfigFile = value

	default:
		return fmt.Errorf("unknown config key: %q", key)
	}

	return nil
}

// validate checks that all required fields are set.
func (c *Config) validate() error {
	if c.MQTTBroker == "" {
		return fmt.Errorf("MQTT_BROKER is required")
	}
	if c.IMULeftSPIDevice == "" {
		return fmt.Errorf("IMU_LEFT_SPI_DEVICE is required")
	}
	if c.IMURightSPIDevice == "" {
		return fmt.Errorf("IMU_RIGHT_SPI_DEVICE is required")
	}
	if c.GPSSerialPort == "" {
		return fmt.Errorf("GPS_SERIAL_PORT is required")
	}
	if c.GPSBaudRate == 0 {
		return fmt.Errorf("GPS_BAUD_RATE is required")
	}
	if c.IMUSampleInterval == 0 {
		return fmt.Errorf("IMU_SAMPLE_INTERVAL is required")
	}
	if c.ConsoleLogInterval == 0 {
		return fmt.Errorf("CONSOLE_LOG_INTERVAL is required")
	}

	// Magnetometer validation with warnings
	if c.MagWriteDelayMS == 0 {
		return fmt.Errorf("MAG_WRITE_DELAY_MS is required")
	}
	if c.MagWriteDelayMS < 50 {
		fmt.Printf("WARNING: MAG_WRITE_DELAY_MS=%dms is below recommended 50ms\n", c.MagWriteDelayMS)
	}
	if c.MagReadDelayMS == 0 {
		return fmt.Errorf("MAG_READ_DELAY_MS is required")
	}
	if c.MagReadDelayMS < 50 {
		fmt.Printf("WARNING: MAG_READ_DELAY_MS=%dms is below recommended 50ms\n", c.MagReadDelayMS)
	}

	// Register debug safety checks
	if c.RegisterDebugMagUnsafeMode {
		fmt.Println("⚠️  REGISTER_DEBUG_MAG_UNSAFE_MODE=true")
		fmt.Println("⚠️  Unsafe magnetometer operations enabled!")
		fmt.Println("⚠️  DO NOT use in production systems")
	}
	if c.RegisterDebugMagWriteDelay > 0 {
		if c.RegisterDebugMagWriteDelay < 50 && !c.RegisterDebugMagUnsafeMode {
			return fmt.Errorf("REGISTER_DEBUG_MAG_WRITE_DELAY < 50ms requires UNSAFE_MODE=true")
		}
		fmt.Printf("⚠️  Using experimental mag write delay: %dms\n", c.RegisterDebugMagWriteDelay)
	}
	if c.RegisterDebugMagReadDelay > 0 {
		if c.RegisterDebugMagReadDelay < 50 && !c.RegisterDebugMagUnsafeMode {
			return fmt.Errorf("REGISTER_DEBUG_MAG_READ_DELAY < 50ms requires UNSAFE_MODE=true")
		}
		fmt.Printf("⚠️  Using experimental mag read delay: %dms\n", c.RegisterDebugMagReadDelay)
	}

	return nil
}

// InitGlobal initializes the global configuration from file.
// Uses sync.Once to ensure this only runs once, even if called multiple times.
// Acquires write lock (configMu.Lock) during initialization to prevent concurrent access.
// This is the only function that can set globalConfig.
func InitGlobal(configPath string) error {
	var err error
	configOnce.Do(func() {
		configMu.Lock()
		defer configMu.Unlock()
		globalConfig, err = Load(configPath)
	})
	return err
}

// Get returns the global configuration instance.
// InitGlobal must be called first, or this will return nil.
// Uses read lock (configMu.RLock) to allow multiple concurrent readers without blocking.
// This is thread-safe and efficient for concurrent access across goroutines.
func Get() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}
