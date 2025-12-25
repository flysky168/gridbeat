package mbus

import (
	"github.com/fluxionwatt/gridbeat/internal/models"
)

// ModbusConfig：单个 modbus 实例的配置
// ModbusConfig: configuration for a single modbus instance.
type InstanceConfig struct {
	Model models.Channel

	URL string `mapstructure:"url"`

	// UnitID 是从站地址（0~247），为 0 时使用库默认值（一般是 1）
	// UnitID is the slave/unit ID (0~247). 0 means "leave library default" (often 1).
	UnitID uint8 `mapstructure:"unit_id"`

	// Timeout 是单次请求超时（例如 "1s"）
	// Timeout is per-request timeout (e.g. "1s").
	//Timeout time.Duration `mapstructure:"timeout"`

	// PollInterval 是轮询周期（例如 "1s"、"500ms"）
	// PollInterval is polling interval (e.g. "1s", "500ms").
	//PollInterval time.Duration `mapstructure:"poll_interval"`

	// StartAddr 是起始寄存器地址
	// StartAddr is the starting register address.
	StartAddr uint16 `mapstructure:"start_addr"`

	// Quantity 是读取的寄存器数量
	// Quantity is the number of registers to read.
	Quantity uint16 `mapstructure:"quantity"`

	// RegType 寄存器类型：
	//   - "holding" => modbus.HOLDING_REGISTER
	//   - "input"   => modbus.INPUT_REGISTER
	//
	// RegType indicates which register type to read:
	//   - "holding" => modbus.HOLDING_REGISTER
	//   - "input"   => modbus.INPUT_REGISTER
	RegType string `mapstructure:"reg_type"`

	//Speed uint
	// DataBits sets the number of bits per serial character (rtu only)
	//DataBits uint
	// Parity sets the serial link parity mode (rtu only)
	//Parity uint
	// StopBits sets the number of serial stop bits (rtu only)
	//StopBits uint
}
