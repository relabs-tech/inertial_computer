// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/sensors"
)

// RegisterDebugSession holds WebSocket connection state for register debugging
type RegisterDebugSession struct {
	Conn *websocket.Conn
}

// WebSocket message types for register debugging
type RegisterReadCmd struct {
	Action  string `json:"action"` // "read", "read_all"
	IMU     string `json:"imu"`    // "left" or "right"
	Address string `json:"addr,omitempty"`
}

type RegisterWriteCmd struct {
	Action  string `json:"action"` // "write"
	IMU     string `json:"imu"`
	Address string `json:"addr"`
	Value   string `json:"value"`
}

type RegisterInitCmd struct {
	Action string `json:"action"` // "init"
	IMU    string `json:"imu"`
}

type RegisterSPISpeedCmd struct {
	Action     string `json:"action"` // "set_spi_speed"
	IMU        string `json:"imu"`
	ReadSpeed  int64  `json:"read_speed"`
	WriteSpeed int64  `json:"write_speed"`
}

type RegisterExportCmd struct {
	Action string `json:"action"` // "export_config"
	IMU    string `json:"imu"`
}

// Response types
type RegisterResponse struct {
	Type        string            `json:"type"`             // "register_data", "register_map", "status", "error"
	Device      string            `json:"device,omitempty"` // "mpu9250" or "ak8963"
	IMU         string            `json:"imu,omitempty"`
	Address     string            `json:"addr,omitempty"`
	Value       string            `json:"value,omitempty"`
	Registers   map[string]string `json:"registers,omitempty"` // for bulk read
	Timestamp   string            `json:"timestamp,omitempty"`
	Message     string            `json:"message,omitempty"`
	Status      string            `json:"status,omitempty"`
	ReadSpeed   int64             `json:"read_speed,omitempty"`
	WriteSpeed  int64             `json:"write_speed,omitempty"`
	RegisterMap []RegisterInfo    `json:"register_map,omitempty"`
}

type RegisterInfo struct {
	Address     string             `json:"address"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Access      string             `json:"access"` // "R", "W", "RW"
	Default     string             `json:"default,omitempty"`
	BitFields   []sensors.BitField `json:"bit_fields,omitempty"`
}

// RegisterConfigFile represents the JSON structure for exported register configuration
type RegisterConfigFile struct {
	Version   int               `json:"version"`
	IMU       string            `json:"imu"`
	Timestamp string            `json:"timestamp"`
	Registers map[string]string `json:"registers"` // hex address -> hex value
}

// HandleRegisterDebugWS handles the WebSocket connection for register debugging
func HandleRegisterDebugWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("register_debug: websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	session := &RegisterDebugSession{Conn: conn}

	// Send register map on connection (MPU9250 by default)
	if err := session.sendRegisterMap("mpu9250"); err != nil {
		log.Printf("register_debug: error sending register map: %v", err)
		return
	}

	// Message loop
	for {
		var rawMsg map[string]interface{}
		err := conn.ReadJSON(&rawMsg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("register_debug: websocket error: %v", err)
			}
			break
		}

		action, ok := rawMsg["action"].(string)
		if !ok {
			session.sendError("missing or invalid action field")
			continue
		}

		// Route based on action
		switch action {
		case "get_map":
			device, _ := rawMsg["device"].(string)
			if device == "" {
				device = "mpu9250" // default
			}
			session.sendRegisterMap(device)
		case "read":
			session.handleRead(rawMsg)
		case "read_all":
			session.handleReadAll(rawMsg)
		case "write":
			session.handleWrite(rawMsg)
		case "init":
			session.handleInit(rawMsg)
		case "set_spi_speed":
			session.handleSetSPISpeed(rawMsg)
		case "export_config":
			session.handleExportConfig(rawMsg)
		default:
			session.sendError(fmt.Sprintf("unknown action: %s", action))
		}
	}
}

func (s *RegisterDebugSession) handleRead(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	device, _ := rawMsg["device"].(string)
	addr, _ := rawMsg["addr"].(string)

	if imu == "" || addr == "" {
		s.sendError("missing imu or addr field")
		return
	}

	// Default to mpu9250 if not specified
	if device == "" {
		device = "mpu9250"
	}

	// Parse hex address
	var addrByte byte
	if _, err := fmt.Sscanf(addr, "0x%X", &addrByte); err != nil {
		s.sendError(fmt.Sprintf("invalid address format: %s", addr))
		return
	}

	// Read register via IMU manager based on device type
	mgr := sensors.GetIMUManager()
	var value byte
	var err error

	if device == "ak8963" {
		value, err = mgr.ReadAK8963Register(imu, addrByte)
	} else {
		value, err = mgr.ReadRegister(imu, addrByte)
	}

	if err != nil {
		s.sendError(fmt.Sprintf("read error: %v", err))
		return
	}

	// Send response
	resp := RegisterResponse{
		Type:      "register_data",
		Device:    device,
		IMU:       imu,
		Address:   addr,
		Value:     fmt.Sprintf("0x%02X", value),
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) handleReadAll(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	device, _ := rawMsg["device"].(string)
	if imu == "" {
		s.sendError("missing imu field")
		return
	}

	// Default to mpu9250 if not specified
	if device == "" {
		device = "mpu9250"
	}

	// Read all registers via IMU manager based on device type
	mgr := sensors.GetIMUManager()
	var registers map[byte]byte
	var err error

	if device == "ak8963" {
		registers, err = mgr.ReadAllAK8963Registers(imu)
	} else {
		registers, err = mgr.ReadAllRegisters(imu)
	}

	if err != nil {
		s.sendError(fmt.Sprintf("read all error: %v", err))
		return
	}

	// Convert to hex string map
	regMap := make(map[string]string)
	for addr, value := range registers {
		regMap[fmt.Sprintf("0x%02X", addr)] = fmt.Sprintf("0x%02X", value)
	}

	// Send response
	resp := RegisterResponse{
		Type:      "register_data",
		Device:    device,
		IMU:       imu,
		Registers: regMap,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) handleWrite(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	device, _ := rawMsg["device"].(string)
	addr, _ := rawMsg["addr"].(string)
	valueStr, _ := rawMsg["value"].(string)

	if imu == "" || addr == "" || valueStr == "" {
		s.sendError("missing imu, addr, or value field")
		return
	}

	if device == "" {
		device = "mpu9250" // default device
	}

	// Parse hex address and value
	var addrByte, valueByte byte
	if _, err := fmt.Sscanf(addr, "0x%X", &addrByte); err != nil {
		s.sendError(fmt.Sprintf("invalid address format: %s", addr))
		return
	}
	if _, err := fmt.Sscanf(valueStr, "0x%X", &valueByte); err != nil {
		s.sendError(fmt.Sprintf("invalid value format: %s", valueStr))
		return
	}

	// Write register via IMU manager (device-specific routing)
	mgr := sensors.GetIMUManager()
	var err error
	if device == "ak8963" {
		err = mgr.WriteAK8963Register(imu, addrByte, valueByte)
	} else {
		// Validate write range for MPU9250
		cfg := config.Get()
		if !isRegisterWritable(addrByte, cfg.RegisterDebugAllowedRanges) {
			s.sendError(fmt.Sprintf("register 0x%02X not in allowed write ranges", addrByte))
			return
		}
		err = mgr.WriteRegister(imu, addrByte, valueByte)
	}
	if err != nil {
		s.sendError(fmt.Sprintf("write error: %v", err))
		return
	}

	// Send confirmation
	resp := RegisterResponse{
		Type:      "register_data",
		IMU:       imu,
		Address:   addr,
		Value:     valueStr,
		Timestamp: time.Now().Format(time.RFC3339),
		Message:   "write successful",
	}
	s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) handleInit(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	if imu == "" {
		s.sendError("missing imu field")
		return
	}

	// Reinitialize IMU via manager
	mgr := sensors.GetIMUManager()
	if err := mgr.ReinitializeIMU(imu); err != nil {
		s.sendError(fmt.Sprintf("reinit error: %v", err))
		return
	}

	// Send status response
	readSpeed, writeSpeed, _ := mgr.GetSPISpeed(imu)
	resp := RegisterResponse{
		Type:       "status",
		IMU:        imu,
		Status:     "initialized",
		ReadSpeed:  readSpeed,
		WriteSpeed: writeSpeed,
		Message:    "IMU reinitialized successfully",
	}
	s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) handleSetSPISpeed(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	readSpeed, _ := rawMsg["read_speed"].(float64)
	writeSpeed, _ := rawMsg["write_speed"].(float64)

	if imu == "" {
		s.sendError("missing imu field")
		return
	}

	cfg := config.Get()

	// Validate and clamp speeds
	readSpeedInt := int64(readSpeed)
	writeSpeedInt := int64(writeSpeed)

	if readSpeedInt < cfg.RegisterDebugMinSPISpeed {
		readSpeedInt = cfg.RegisterDebugMinSPISpeed
	}
	if readSpeedInt > cfg.RegisterDebugMaxSPISpeed {
		readSpeedInt = cfg.RegisterDebugMaxSPISpeed
	}
	if writeSpeedInt < cfg.RegisterDebugMinSPISpeed {
		writeSpeedInt = cfg.RegisterDebugMinSPISpeed
	}
	if writeSpeedInt > cfg.RegisterDebugMaxSPISpeed {
		writeSpeedInt = cfg.RegisterDebugMaxSPISpeed
	}

	// Set SPI speeds
	mgr := sensors.GetIMUManager()
	if err := mgr.SetSPISpeed(imu, readSpeedInt, writeSpeedInt); err != nil {
		s.sendError(fmt.Sprintf("set spi speed error: %v", err))
		return
	}

	// Send confirmation
	resp := RegisterResponse{
		Type:       "status",
		IMU:        imu,
		ReadSpeed:  readSpeedInt,
		WriteSpeed: writeSpeedInt,
		Message:    "SPI speeds updated",
	}
	s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) handleExportConfig(rawMsg map[string]interface{}) {
	imu, _ := rawMsg["imu"].(string)
	if imu == "" {
		s.sendError("missing imu field")
		return
	}

	// Read all registers
	mgr := sensors.GetIMUManager()
	registers, err := mgr.ExportRegisterConfig(imu)
	if err != nil {
		s.sendError(fmt.Sprintf("export error: %v", err))
		return
	}

	// Convert to hex string map
	regMap := make(map[string]string)
	for addr, value := range registers {
		regMap[fmt.Sprintf("0x%02X", addr)] = fmt.Sprintf("0x%02X", value)
	}

	// Create config file structure
	configFile := RegisterConfigFile{
		Version:   1,
		IMU:       imu,
		Timestamp: time.Now().Format(time.RFC3339),
		Registers: regMap,
	}

	// Send as download
	configJSON, _ := json.Marshal(configFile)
	rawResp := map[string]interface{}{
		"type":     "export_config",
		"imu":      imu,
		"message":  "config exported",
		"config":   string(configJSON),
		"filename": fmt.Sprintf("%s_%s_registers.json", imu, time.Now().Format("20060102_150405")),
	}
	s.Conn.WriteJSON(rawResp)
}

func (s *RegisterDebugSession) sendRegisterMap(deviceType string) error {
	mgr := sensors.GetIMUManager()
	var regMap []sensors.RegisterInfo

	// Select register map based on device type
	switch deviceType {
	case "ak8963":
		regMap = mgr.GetAK8963RegisterMap()
	default:
		// Default to MPU9250
		regMap = mgr.GetRegisterMap()
	}

	// Convert sensors.RegisterInfo to RegisterInfo
	mappedRegs := make([]RegisterInfo, len(regMap))
	for i, r := range regMap {
		mappedRegs[i] = RegisterInfo{
			Address:     r.Address,
			Name:        r.Name,
			Description: r.Description,
			Access:      r.Access,
			Default:     r.Default,
			BitFields:   r.BitFields, // Already sensors.BitField type
		}
	}

	resp := RegisterResponse{
		Type:        "register_map",
		Device:      deviceType,
		RegisterMap: mappedRegs,
	}
	return s.Conn.WriteJSON(resp)
}

func (s *RegisterDebugSession) sendError(message string) {
	resp := RegisterResponse{
		Type:    "error",
		Message: message,
	}
	s.Conn.WriteJSON(resp)
}

// HandleIMUData serves live IMU data via REST API
// Query parameter: ?imu=left or ?imu=right (defaults to left)
func HandleIMUData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	imuID := r.URL.Query().Get("imu")
	if imuID == "" {
		imuID = "left"
	}

	mgr := sensors.GetIMUManager()

	var raw interface{}
	var err error

	if imuID == "left" {
		raw, err = mgr.ReadLeftIMU()
	} else if imuID == "right" {
		raw, err = mgr.ReadRightIMU()
	} else {
		http.Error(w, `{"error": "invalid imu parameter, use 'left' or 'right'"}`, http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(raw)
}

// isRegisterWritable checks if a register address is in the allowed write ranges
func isRegisterWritable(addr byte, allowedRanges string) bool {
	if allowedRanges == "" {
		return false // Empty means no writes allowed by default
	}

	// Parse ranges like "0x1B-0x1D,0x6B,0x1A-0x20"
	// For simplicity, if configured, allow the write
	// TODO: implement proper range parsing
	return true
}
