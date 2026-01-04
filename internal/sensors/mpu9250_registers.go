// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

package sensors

// getMPU9250RegisterMap returns metadata for all MPU9250 registers.
// This provides register names, descriptions, access types, and bit field definitions.
func getMPU9250RegisterMap() []RegisterInfo {
	return []RegisterInfo{
		// Configuration Registers
		{Address: "0x19", Name: "SMPLRT_DIV", Description: "Sample Rate Divider", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "SMPLRT_DIV", Description: "Sample Rate = Internal_Sample_Rate / (1 + SMPLRT_DIV)", Values: "0-255"},
			}},
		{Address: "0x1A", Name: "CONFIG", Description: "Configuration (DLPF)", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "6", Name: "FIFO_MODE", Description: "FIFO mode", Values: "0=Overwrite, 1=Block new data"},
				{Bits: "5:3", Name: "EXT_SYNC_SET", Description: "External FSYNC pin sampling", Values: "0=Disabled"},
				{Bits: "2:0", Name: "DLPF_CFG", Description: "Digital Low Pass Filter", Values: "0=250Hz, 1=184Hz, 2=92Hz, 3=41Hz, 4=20Hz, 5=10Hz, 6=5Hz, 7=3600Hz"},
			}},
		{Address: "0x1B", Name: "GYRO_CONFIG", Description: "Gyroscope Configuration", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "XGYRO_Cten", Description: "X Gyro self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "6", Name: "YGYRO_Cten", Description: "Y Gyro self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "5", Name: "ZGYRO_Cten", Description: "Z Gyro self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "4:3", Name: "GYRO_FS_SEL", Description: "Gyro Full Scale Range", Values: "0=±250°/s, 1=±500°/s, 2=±1000°/s, 3=±2000°/s"},
				{Bits: "1:0", Name: "Fchoice_b", Description: "Gyro DLPF bypass", Values: "0=DLPF enabled"},
			}},
		{Address: "0x1C", Name: "ACCEL_CONFIG", Description: "Accelerometer Configuration", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "ax_st_en", Description: "X Accel self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "6", Name: "ay_st_en", Description: "Y Accel self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "5", Name: "az_st_en", Description: "Z Accel self-test", Values: "0=Disabled, 1=Enabled"},
				{Bits: "4:3", Name: "ACCEL_FS_SEL", Description: "Accel Full Scale Range", Values: "0=±2g, 1=±4g, 2=±8g, 3=±16g"},
			}},
		{Address: "0x1D", Name: "ACCEL_CONFIG2", Description: "Accelerometer Configuration 2", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "3", Name: "accel_fchoice_b", Description: "Accel DLPF bypass", Values: "0=DLPF enabled, 1=Bypass"},
				{Bits: "2:0", Name: "A_DLPFCFG", Description: "Accel DLPF Config", Values: "0=460Hz, 1=184Hz, 2=92Hz, 3=41Hz, 4=20Hz, 5=10Hz, 6=5Hz, 7=460Hz"},
			}},
		{Address: "0x1E", Name: "LP_ACCEL_ODR", Description: "Low Power Accelerometer ODR Control", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "3:0", Name: "Lposc_clksel", Description: "Low Power Accel Output Data Rate", Values: "0=0.24Hz ... 11=500Hz"},
			}},

		// Interrupt Configuration
		{Address: "0x37", Name: "INT_PIN_CFG", Description: "INT Pin / Bypass Enable Configuration", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "ACTL", Description: "INT pin active low", Values: "0=Active high, 1=Active low"},
				{Bits: "6", Name: "OPEN", Description: "INT pin open drain", Values: "0=Push-pull, 1=Open drain"},
				{Bits: "5", Name: "LATCH_INT_EN", Description: "Latch INT pin", Values: "0=50us pulse, 1=Latch until cleared"},
				{Bits: "4", Name: "INT_ANYRD_2CLEAR", Description: "Clear INT on any read", Values: "0=Status read only, 1=Any read"},
				{Bits: "3", Name: "ACTL_FSYNC", Description: "FSYNC pin active low", Values: "0=Active high, 1=Active low"},
				{Bits: "2", Name: "FSYNC_INT_MODE_EN", Description: "Enable FSYNC as interrupt", Values: "0=Disabled, 1=Enabled"},
				{Bits: "1", Name: "BYPASS_EN", Description: "I2C bypass enable", Values: "0=Disabled, 1=Enabled"},
			}},
		{Address: "0x38", Name: "INT_ENABLE", Description: "Interrupt Enable", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "6", Name: "WOM_EN", Description: "Wake on Motion interrupt", Values: "0=Disabled, 1=Enabled"},
				{Bits: "4", Name: "FIFO_OVERFLOW_EN", Description: "FIFO overflow interrupt", Values: "0=Disabled, 1=Enabled"},
				{Bits: "3", Name: "FSYNC_INT_EN", Description: "FSYNC interrupt", Values: "0=Disabled, 1=Enabled"},
				{Bits: "0", Name: "RAW_RDY_EN", Description: "Raw data ready interrupt", Values: "0=Disabled, 1=Enabled"},
			}},
		{Address: "0x3A", Name: "INT_STATUS", Description: "Interrupt Status", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "6", Name: "WOM_INT", Description: "Wake on Motion interrupt status", Values: ""},
				{Bits: "4", Name: "FIFO_OVERFLOW_INT", Description: "FIFO overflow interrupt status", Values: ""},
				{Bits: "3", Name: "FSYNC_INT", Description: "FSYNC interrupt status", Values: ""},
				{Bits: "0", Name: "RAW_DATA_RDY_INT", Description: "Raw data ready interrupt status", Values: ""},
			}},

		// Sensor Data Registers (Read-Only)
		{Address: "0x3B", Name: "ACCEL_XOUT_H", Description: "Accelerometer X-Axis High Byte", Access: "R"},
		{Address: "0x3C", Name: "ACCEL_XOUT_L", Description: "Accelerometer X-Axis Low Byte", Access: "R"},
		{Address: "0x3D", Name: "ACCEL_YOUT_H", Description: "Accelerometer Y-Axis High Byte", Access: "R"},
		{Address: "0x3E", Name: "ACCEL_YOUT_L", Description: "Accelerometer Y-Axis Low Byte", Access: "R"},
		{Address: "0x3F", Name: "ACCEL_ZOUT_H", Description: "Accelerometer Z-Axis High Byte", Access: "R"},
		{Address: "0x40", Name: "ACCEL_ZOUT_L", Description: "Accelerometer Z-Axis Low Byte", Access: "R"},
		{Address: "0x41", Name: "TEMP_OUT_H", Description: "Temperature High Byte", Access: "R"},
		{Address: "0x42", Name: "TEMP_OUT_L", Description: "Temperature Low Byte", Access: "R"},
		{Address: "0x43", Name: "GYRO_XOUT_H", Description: "Gyroscope X-Axis High Byte", Access: "R"},
		{Address: "0x44", Name: "GYRO_XOUT_L", Description: "Gyroscope X-Axis Low Byte", Access: "R"},
		{Address: "0x45", Name: "GYRO_YOUT_H", Description: "Gyroscope Y-Axis High Byte", Access: "R"},
		{Address: "0x46", Name: "GYRO_YOUT_L", Description: "Gyroscope Y-Axis Low Byte", Access: "R"},
		{Address: "0x47", Name: "GYRO_ZOUT_H", Description: "Gyroscope Z-Axis High Byte", Access: "R"},
		{Address: "0x48", Name: "GYRO_ZOUT_L", Description: "Gyroscope Z-Axis Low Byte", Access: "R"},

		// External Sensor Data (Magnetometer via I2C)
		{Address: "0x49", Name: "EXT_SENS_DATA_00", Description: "External Sensor Data 00", Access: "R"},
		{Address: "0x4A", Name: "EXT_SENS_DATA_01", Description: "External Sensor Data 01", Access: "R"},
		{Address: "0x4B", Name: "EXT_SENS_DATA_02", Description: "External Sensor Data 02", Access: "R"},
		{Address: "0x4C", Name: "EXT_SENS_DATA_03", Description: "External Sensor Data 03", Access: "R"},
		{Address: "0x4D", Name: "EXT_SENS_DATA_04", Description: "External Sensor Data 04", Access: "R"},
		{Address: "0x4E", Name: "EXT_SENS_DATA_05", Description: "External Sensor Data 05", Access: "R"},
		{Address: "0x4F", Name: "EXT_SENS_DATA_06", Description: "External Sensor Data 06", Access: "R"},

		// I2C Master Control
		{Address: "0x23", Name: "I2C_MST_CTRL", Description: "I2C Master Control", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "MULT_MST_EN", Description: "Multi-master enable", Values: "0=Disabled, 1=Enabled"},
				{Bits: "4", Name: "WAIT_FOR_ES", Description: "Wait for external sensor", Values: "0=Disabled, 1=Enabled"},
				{Bits: "3:0", Name: "I2C_MST_CLK", Description: "I2C Master clock speed", Values: "0=348kHz ... 15=24kHz"},
			}},
		{Address: "0x24", Name: "I2C_SLV0_ADDR", Description: "I2C Slave 0 Address", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "I2C_SLV0_RNW", Description: "Read/Write mode", Values: "0=Write, 1=Read"},
				{Bits: "6:0", Name: "I2C_ID_0", Description: "I2C slave address", Values: "7-bit address"},
			}},
		{Address: "0x25", Name: "I2C_SLV0_REG", Description: "I2C Slave 0 Register", Access: "RW", Default: "0x00"},
		{Address: "0x26", Name: "I2C_SLV0_CTRL", Description: "I2C Slave 0 Control", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7", Name: "I2C_SLV0_EN", Description: "Enable reading", Values: "0=Disabled, 1=Enabled"},
				{Bits: "6", Name: "I2C_SLV0_BYTE_SW", Description: "Byte swap", Values: "0=No swap, 1=Swap"},
				{Bits: "5", Name: "I2C_SLV0_REG_DIS", Description: "Register disable", Values: ""},
				{Bits: "4", Name: "I2C_SLV0_GRP", Description: "Group registers", Values: ""},
				{Bits: "3:0", Name: "I2C_SLV0_LENG", Description: "Number of bytes to read", Values: "0-15"},
			}},

		// FIFO Configuration
		{Address: "0x23", Name: "FIFO_EN", Description: "FIFO Enable", Access: "RW", Default: "0x00"},
		{Address: "0x6A", Name: "USER_CTRL", Description: "User Control", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "6", Name: "FIFO_EN", Description: "Enable FIFO", Values: "0=Disabled, 1=Enabled"},
				{Bits: "5", Name: "I2C_MST_EN", Description: "Enable I2C Master", Values: "0=Disabled, 1=Enabled"},
				{Bits: "4", Name: "I2C_IF_DIS", Description: "Disable I2C Slave", Values: "0=Enabled, 1=Disabled"},
				{Bits: "2", Name: "FIFO_RST", Description: "Reset FIFO", Values: "1=Reset"},
				{Bits: "1", Name: "I2C_MST_RST", Description: "Reset I2C Master", Values: "1=Reset"},
				{Bits: "0", Name: "SIG_COND_RST", Description: "Reset signal paths", Values: "1=Reset"},
			}},
		{Address: "0x6B", Name: "PWR_MGMT_1", Description: "Power Management 1", Access: "RW", Default: "0x01",
			BitFields: []BitField{
				{Bits: "7", Name: "H_RESET", Description: "Device reset", Values: "1=Reset device"},
				{Bits: "6", Name: "SLEEP", Description: "Sleep mode", Values: "0=Disabled, 1=Sleep"},
				{Bits: "5", Name: "CYCLE", Description: "Cycle mode", Values: "0=Disabled, 1=Cycle"},
				{Bits: "3", Name: "TEMP_DIS", Description: "Temperature sensor", Values: "0=Enabled, 1=Disabled"},
				{Bits: "2:0", Name: "CLKSEL", Description: "Clock source", Values: "0=Internal 20MHz, 1=Auto select best"},
			}},
		{Address: "0x6C", Name: "PWR_MGMT_2", Description: "Power Management 2", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "5", Name: "DISABLE_XA", Description: "Disable X accelerometer", Values: "0=Enabled, 1=Disabled"},
				{Bits: "4", Name: "DISABLE_YA", Description: "Disable Y accelerometer", Values: "0=Enabled, 1=Disabled"},
				{Bits: "3", Name: "DISABLE_ZA", Description: "Disable Z accelerometer", Values: "0=Enabled, 1=Disabled"},
				{Bits: "2", Name: "DISABLE_XG", Description: "Disable X gyro", Values: "0=Enabled, 1=Disabled"},
				{Bits: "1", Name: "DISABLE_YG", Description: "Disable Y gyro", Values: "0=Enabled, 1=Disabled"},
				{Bits: "0", Name: "DISABLE_ZG", Description: "Disable Z gyro", Values: "0=Enabled, 1=Disabled"},
			}},
		{Address: "0x72", Name: "FIFO_COUNTH", Description: "FIFO Count High Byte", Access: "R"},
		{Address: "0x73", Name: "FIFO_COUNTL", Description: "FIFO Count Low Byte", Access: "R"},
		{Address: "0x74", Name: "FIFO_R_W", Description: "FIFO Read Write", Access: "RW"},

		// Device Identification
		{Address: "0x75", Name: "WHO_AM_I", Description: "Device ID (should be 0x71)", Access: "R", Default: "0x71"},

		// Accelerometer Offset Trim
		{Address: "0x77", Name: "XA_OFFSET_H", Description: "Accelerometer X-Axis Offset High Byte", Access: "RW"},
		{Address: "0x78", Name: "XA_OFFSET_L", Description: "Accelerometer X-Axis Offset Low Byte", Access: "RW"},
		{Address: "0x7A", Name: "YA_OFFSET_H", Description: "Accelerometer Y-Axis Offset High Byte", Access: "RW"},
		{Address: "0x7B", Name: "YA_OFFSET_L", Description: "Accelerometer Y-Axis Offset Low Byte", Access: "RW"},
		{Address: "0x7D", Name: "ZA_OFFSET_H", Description: "Accelerometer Z-Axis Offset High Byte", Access: "RW"},
		{Address: "0x7E", Name: "ZA_OFFSET_L", Description: "Accelerometer Z-Axis Offset Low Byte", Access: "RW"},
	}
}

// getAK8963RegisterMap returns metadata for all AK8963 magnetometer registers.
// The AK8963 is accessed via MPU9250's internal I2C master using EXT_SENS_DATA registers.
// Accessed via I2C slave address 0x0C.
func getAK8963RegisterMap() []RegisterInfo {
	return []RegisterInfo{
		// Identification and Status
		{Address: "0x00", Name: "WIA", Description: "WHO_AM_I - Device identification (should be 0x48)", Access: "R", Default: "0x48",
			BitFields: []BitField{
				{Bits: "7:0", Name: "WIA", Description: "Device ID", Values: "0x48=AK8963"},
			}},
		{Address: "0x01", Name: "INFO", Description: "INFO - Information about device", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "INFO", Description: "Information register", Values: "Varies by device"},
			}},

		// Data Status and Readings
		{Address: "0x02", Name: "ST1", Description: "STATUS 1 - Data ready and overrun status", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "0", Name: "DRDY", Description: "Data Ready", Values: "0=Not ready, 1=Data ready"},
				{Bits: "1", Name: "DOR", Description: "Data Overrun", Values: "0=No overrun, 1=Data overrun"},
				{Bits: "7:2", Name: "RESERVED", Description: "Reserved", Values: "Always 0"},
			}},
		{Address: "0x03", Name: "HXL", Description: "X-AXIS DATA LOW - Magnetometer X low byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HX[7:0]", Description: "X-axis data low byte", Values: "0-255"},
			}},
		{Address: "0x04", Name: "HXH", Description: "X-AXIS DATA HIGH - Magnetometer X high byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HX[15:8]", Description: "X-axis data high byte", Values: "0-255"},
			}},
		{Address: "0x05", Name: "HYL", Description: "Y-AXIS DATA LOW - Magnetometer Y low byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HY[7:0]", Description: "Y-axis data low byte", Values: "0-255"},
			}},
		{Address: "0x06", Name: "HYH", Description: "Y-AXIS DATA HIGH - Magnetometer Y high byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HY[15:8]", Description: "Y-axis data high byte", Values: "0-255"},
			}},
		{Address: "0x07", Name: "HZL", Description: "Z-AXIS DATA LOW - Magnetometer Z low byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HZ[7:0]", Description: "Z-axis data low byte", Values: "0-255"},
			}},
		{Address: "0x08", Name: "HZH", Description: "Z-AXIS DATA HIGH - Magnetometer Z high byte", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "HZ[15:8]", Description: "Z-axis data high byte", Values: "0-255"},
			}},
		{Address: "0x09", Name: "ST2", Description: "STATUS 2 - Data status and overflow check", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "3", Name: "HOFL", Description: "Magnetic Sensor Overflow", Values: "0=No overflow, 1=Data overflow occurred"},
				{Bits: "4", Name: "BITM", Description: "Output Data Bit Width", Values: "0=14-bit, 1=16-bit resolution"},
				{Bits: "7:5", Name: "RESERVED", Description: "Reserved", Values: "Always 0"},
			}},

		// Control Registers
		{Address: "0x0A", Name: "CNTL1", Description: "CONTROL 1 - Operation mode and resolution", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "3:0", Name: "MODE", Description: "Operation Mode", Values: "0=PowerDown, 1=SingleMeasure, 2=Continuous1(10Hz), 6=Continuous2(100Hz), 8=ExternalTrigger, 15=SelfTest"},
				{Bits: "4", Name: "BIT", Description: "Output Data Bit Width", Values: "0=14-bit, 1=16-bit"},
				{Bits: "7:5", Name: "RESERVED", Description: "Reserved", Values: "Always 0"},
			}},
		{Address: "0x0B", Name: "CNTL2", Description: "CONTROL 2 - Soft reset", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "0", Name: "SRST", Description: "Soft Reset", Values: "0=Normal, 1=Reset magnetometer"},
				{Bits: "7:1", Name: "RESERVED", Description: "Reserved", Values: "Always 0"},
			}},
		{Address: "0x0C", Name: "ASTC", Description: "ASTC - Self-test control", Access: "RW", Default: "0x00",
			BitFields: []BitField{
				{Bits: "6", Name: "SELF", Description: "Self-Test Enable", Values: "0=Disabled, 1=Generate magnetic field for self-test"},
				{Bits: "7", Name: "RESERVED", Description: "Reserved", Values: "Always 0"},
			}},

		// Factory Calibration (Sensitivity Adjustment)
		{Address: "0x10", Name: "ASAX", Description: "X-AXIS SENSITIVITY ADJUST - Factory calibration for X", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "ASAX", Description: "X-axis sensitivity adjustment", Values: "Applied as: (ASA[0]-128)/256 + 1.0"},
			}},
		{Address: "0x11", Name: "ASAY", Description: "Y-AXIS SENSITIVITY ADJUST - Factory calibration for Y", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "ASAY", Description: "Y-axis sensitivity adjustment", Values: "Applied as: (ASA[1]-128)/256 + 1.0"},
			}},
		{Address: "0x12", Name: "ASAZ", Description: "Z-AXIS SENSITIVITY ADJUST - Factory calibration for Z", Access: "R", Default: "0x00",
			BitFields: []BitField{
				{Bits: "7:0", Name: "ASAZ", Description: "Z-axis sensitivity adjustment", Values: "Applied as: (ASA[2]-128)/256 + 1.0"},
			}},
	}
}
