package models

type RegType string

const (
	RegHolding  RegType = "holding"  // 0x03 / 0x06 / 0x10
	RegInput    RegType = "input"    // 0x04
	RegCoil     RegType = "coil"     // 0x01 / 0x05 / 0x0F
	RegDiscrete RegType = "discrete" // 0x02
)

type RegAccess string

const (
	RegRO RegAccess = "RO"
	RegRW RegAccess = "RW"
	RegWO RegAccess = "WO"
)

type Device struct {
	ChannelID string
	Channel   Channel `gorm:"references:UUID"`
	Base
	Name            string `gorm:"column:name;size:150;uniqueIndex;not null" json:"name"`
	DeviceType      string `gorm:"column:device_type;size:128;not null;index" json:"device_type"` // Requirement #4: weak association by type_key (no FK)
	Transport       string `gorm:"size:16;not null"`                                              // tcp/rtu/rtu_over_tcp...
	Endpoint        string `gorm:"size:256;not null"`                                             // host:port or /dev/ttyS1
	SlaveID         int    `gorm:"not null;default:1"`
	PollIntervalMs  int    `gorm:"not null;default:1000"`
	SN              string `gorm:"column:sn;size:128;not null" json:"sn"`
	DevicePlugin    string `gorm:"column:device_plugin;size:128;not null" json:"device_plugin"`
	SoftwareVersion string `gorm:"column:software_version;size:128;not null" json:"software_version"`
	Model           string `gorm:"column:model;size:128;not null" json:"model"`
	Disable         bool   `gorm:"column:disable;size:128;not null" json:"disable"`
}

type DeviceState struct {
	Online bool `gorm:"-" json:"online"`
}

// TableName 用来显式指定表名（可选）
func (Device) TableName() string {
	return "device"
}

type DeveltypePointBase struct {
	Name      string  `gorm:"size:128;;not null"`                // 信号名
	NameCn    string  `gorm:"size:128;;not null"`                // 信号名
	Gain      float64 `gorm:"not null;default:0"`                // 文档里的 Gain
	Unit      string  `gorm:"size:32"`                           // 单位（V, A, kWh, %...）
	RW        string  `gorm:"size:8;not null;default:'RO'"`      // RO / RW / WO
	Address   uint16  `gorm:"not null"`                          // Modbus 地址（手册里的 Address）
	PointType string  `gorm:"size:8;not null;default:'holding'"` // 点位类型
	Quantity  uint32  `gorm:"not null;default:1"`                // 增益
	DataType  string  `gorm:"size:32;not null"`                  // int16/uint16/int32/float32/bool/string...
	ByteOrder string  `gorm:"size:8"`                            // ABCD/BADC/CDAB/DCBA...
	Offset    float64 `gorm:"not null;default:0"`                // offset
	Desc      string  `gorm:"size:128"`                          // Desc
	Disable   bool    `gorm:"not null;default:false"`            // such as
}

type DeviceTypePoint struct {
	Base
	TypeKey string `gorm:"size:128;uniqueIndex;not null"`
	Model   string `gorm:"size:128"`           // 逆变器/STS
	Index   uint   `json:"index" yaml:"index"` // 点位 index
	DeveltypePointBase
}

// TableName 用来显式指定表名（可选）
func (DeviceTypePoint) TableName() string {
	return "device_type"
}

// Precision     int     `gorm:"not null;default:0"`
// EnumMapJSON   []byte  `gorm:"type:json"` // optional enums, e.g. {"0":"Off","1":"On"}
// Scale         float64 `gorm:"not null;default:1"`
