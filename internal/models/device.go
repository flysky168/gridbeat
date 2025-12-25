package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type RegType string

const (
	RegHolding  RegType = "holding"  // 0x03 / 0x06 / 0x10
	RegInput    RegType = "input"    // 0x04
	RegCoil     RegType = "coil"     // 0x01 / 0x05 / 0x0F
	RegDiscrete RegType = "discrete" // 0x02
)

type RegDataType string

const (
	RegU16     RegDataType = "u16"
	RegS16     RegDataType = "s16"
	RegU32     RegDataType = "u32"
	RegS32     RegDataType = "s32"
	RegFloat32 RegDataType = "float32"
	RegFloat64 RegDataType = "float64"
	RegBitMask RegDataType = "bitmask"
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

type RegisterDef struct {
	Addr     uint16  // Modbus 地址（手册里的 Address）
	Quantity uint16  // Quantity
	Name     string  // 信号名（用于调试）
	T        string  // 寄存器类型
	Gain     float64 // 文档里的 Gain
	Unit     string  // 单位（V, A, kWh, %...）
	RW       string  // RO / RW / WO
	Desc     string
}

type DeviceType struct {
	ID        string `gorm:"primaryKey;type:char(36)"`
	TypeKey   string `gorm:"size:128;uniqueIndex;not null"`
	Name      string `gorm:"size:256;not null"`
	Vendor    string `gorm:"size:128;index"`
	Model     string `gorm:"size:128;index"`
	Version   string `gorm:"size:64"`
	Checksum  string `gorm:"size:64"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// I18nMap is stored as a single JSON column, no extra lang table.
// English key "en" is required by validation.
type I18nMap map[string]string

func (m I18nMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *I18nMap) Scan(src any) error {
	if m == nil {
		return fmt.Errorf("I18nMap.Scan: nil receiver")
	}
	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			*m = I18nMap{}
			return nil
		}
		return json.Unmarshal(v, m)
	case string:
		if v == "" {
			*m = I18nMap{}
			return nil
		}
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("I18nMap.Scan: unsupported type %T", src)
	}
}

type DeviceTypePoint struct {
	ID        string  `gorm:"primaryKey;type:char(36)"`
	TypeKey   string  `gorm:"size:128;not null;index;uniqueIndex:uniq_type_point"`
	PointCode string  `gorm:"size:128;not null;uniqueIndex:uniq_type_point"`
	PointKind RegType `gorm:"size:2;not null;index"`

	// Requirement #7: vendor/model fields must be in device_type_points.
	Vendor string `gorm:"size:128;index"`
	Model  string `gorm:"size:128;index"`

	// Requirement #9: English name required, optional Chinese, extensible languages, no lang table.
	NameI18n I18nMap `gorm:"type:json;not null"`

	Unit    string `gorm:"size:32"`
	RW      string `gorm:"size:8;not null;default:'R'"`
	Enabled bool   `gorm:"not null;default:true"`

	// Modbus mapping (Requirement #6: one point -> one modbus signal)
	FC       uint8  `gorm:"not null"` // 1/2/3/4/5/6/15/16
	Address  uint16 `gorm:"not null"`
	Quantity uint16 `gorm:"not null;default:1"`
	DataType string `gorm:"size:32;not null"` // int16/uint16/int32/float32/bool/string...
	BitIndex *uint8 `gorm:""`                 // optional (bit in register/bitfield)

	ByteOrder string  `gorm:"size:8"` // ABCD/BADC/CDAB/DCBA...
	Scale     float64 `gorm:"not null;default:1"`
	Offset    float64 `gorm:"not null;default:0"`
	Precision int     `gorm:"not null;default:0"`

	EnumMapJSON []byte `gorm:"type:json"` // optional enums, e.g. {"0":"Off","1":"On"}

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type ModbusSpec struct {
	FC        uint8   `json:"fc" yaml:"fc"`
	Address   uint16  `json:"address" yaml:"address"`
	Quantity  uint16  `json:"quantity" yaml:"quantity"`
	DataType  string  `json:"data_type" yaml:"data_type"`
	BitIndex  *uint8  `json:"bit_index" yaml:"bit_index"`
	ByteOrder string  `json:"byte_order" yaml:"byte_order"`
	Scale     float64 `json:"scale" yaml:"scale"`
	Offset    float64 `json:"offset" yaml:"offset"`
	Precision int     `json:"precision" yaml:"precision"`
	EnumMap   any     `json:"enum_map" yaml:"enum_map"` // optional; importer will json-marshal
}

type PointSpec struct {
	Code     string     `json:"code" yaml:"code"`
	Kind     RegType    `json:"kind" yaml:"kind"`
	RW       string     `json:"rw" yaml:"rw"`
	Unit     string     `json:"unit" yaml:"unit"`
	NameI18n I18nMap    `json:"name_i18n" yaml:"name_i18n"`
	Modbus   ModbusSpec `json:"modbus" yaml:"modbus"`
}

type TypeSpec struct {
	TypeKey string `json:"type_key" yaml:"type_key"`
	NameEn  string `json:"name_en" yaml:"name_en"`
	Vendor  string `json:"vendor" yaml:"vendor"`
	Model   string `json:"model" yaml:"model"`
	Version string `json:"version" yaml:"version"`

	Points []PointSpec `json:"points" yaml:"points"`
}
