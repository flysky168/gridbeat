package models

import (
	"time"

	"github.com/fluxionwatt/gridbeat/utils/modbus"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChannelStatus struct {
	Working         bool          `gorm:"-" json:"working"`           // 工作状态
	Linking         bool          `gorm:"-" json:"linking"`           // 连接状态
	CurrentDelay    time.Duration `gorm:"-" json:"current_delay"`     // 当前采集延迟
	BytesSent       uint64        `gorm:"-" json:"bytes_sent"`        // bytes sent
	BytesReceived   uint64        `gorm:"-" json:"bytes_received"`    // bytes received
	PointsToalRead  uint64        `gorm:"-" json:"points_total_read"` // 点位读取数总计
	PointsErrorRead uint64        `gorm:"-" json:"points_error_read"` // 点位读取错误数总计
}

// Channel 通道
type Channel struct {
	ID            uint          `gorm:"primaryKey;autoIncrement" json:"id"`          // ID is auto-increment primary key / 自增主键
	Delay         time.Duration `gorm:"column:delay" json:"delay"`                   // 时延 单位 毫秒
	DebugLog      bool          `gorm:"column:debug_log" json:"debug_log"`           // debug log 模式
	DebugExpiry   time.Time     `gorm:"column:debug_expiry" json:"debug_expiry"`     // debuglog 截止时间
	VerifyHeader  bool          `gorm:"column:verify_header" json:"verify_header"`   // 校验报文头
	OnnectTimeout time.Duration `gorm:"column:onnect_timeout" json:"onnect_timeout"` // 连接超时时间 (ms)，默认 3s
	Downgrade     bool          `gorm:"column:down_grade" json:"down_grade"`         // 设备降级
	RetryMax      uint64        `gorm:"column:retry_max" json:"retry_max"`           // 最大重试次数,发送读指令失败后最大重试次数
	RetryInterval time.Duration `gorm:"column:retry_interval" json:"retry_interval"` // 指令重新发送间隔 (ms)
	Endianness    uint          `gorm:"column:endianness" json:"endianness"`         // endianness
	WordOrder     uint          `gorm:"column:word_order" json:"word_order"`         // word ordering
	SendInterval  time.Duration `gorm:"column:send_interval" json:"send_interval"`   // 指令发送间隔 (ms)
	PhysicalLink  string        `gorm:"column:physical_link" json:"physical_link"`   // serial、RTUclient、RTUserver、TCPclient、TCPserver
	UUID          string        `gorm:"column:uuid;size:36;uniqueIndex;not null" json:"uuid"`
	Disable       bool          `gorm:"column:disable" json:"disable"` // disable
	Plugin        string        `gorm:"column:plugin;size:128;not null" json:"plugin"`

	// Serial
	Device   string `gorm:"uniqueIndex;size:256;not null" json:"device"`  // Device is like "/dev/ttyUSB0" / 设备路径，例如 "/dev/ttyUSB0"
	Device2  string `gorm:"uniqueIndex;size:256;not null" json:"device2"` // Device is like "/dev/ttyUSB0" / 设备路径，例如 "/dev/ttyUSB1"
	StopBits uint   `gorm:"column:stop_bits" json:"stop_bits"`            // 停止位
	Speed    uint   `gorm:"column:speed" json:"speed"`                    // speed
	DataBits uint   `gorm:"column:data_bits" json:"data_bits"`            // data bits
	Parity   uint   `gorm:"column:parity" json:"parity"`                  // parity
	// TCP
	AddrStart       bool
	TCPIPAddr       string
	TCPPort         uint16
	BackupTCPIPAddr string
	BackupTCPPort   uint16

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Status ChannelStatus `gorm:"-" json:"status"`
}

// TableName 用来显式指定表名（可选）
func (Channel) TableName() string {
	return "channel"
}

func GetDefaultSerialRow(device, device2 string) *Channel {
	return &Channel{
		UUID:            uuid.New().String(),
		Plugin:          "mbus",
		PhysicalLink:    "serial",
		Device:          device,
		Device2:         device2,
		Delay:           300 * time.Millisecond,
		DebugLog:        false,
		DebugExpiry:     time.Now(),
		VerifyHeader:    false,
		Speed:           19200,
		OnnectTimeout:   300 * time.Millisecond,
		Downgrade:       false,
		RetryMax:        3,
		RetryInterval:   300 * time.Millisecond,
		Endianness:      uint(modbus.LITTLE_ENDIAN),
		WordOrder:       uint(modbus.LOW_WORD_FIRST),
		SendInterval:    300 * time.Millisecond,
		StopBits:        2,
		DataBits:        8,
		Parity:          modbus.PARITY_NONE,
		TCPIPAddr:       "localhost",
		TCPPort:         5502,
		BackupTCPIPAddr: "localhost",
		BackupTCPPort:   5502,
		Disable:         false,
	}
}

/*
GORM（v2）里增删改相关的 model hook 主要就是这些（方法签名统一为 func(*gorm.DB) error，返回 error 会中止后续并回滚当前事务）。 ￼

Create（新增）会触发
  - BeforeSave
  - BeforeCreate
  - AfterCreate
  - AfterSave  ￼

Update（修改）会触发
  - BeforeSave
  - BeforeUpdate
  - AfterUpdate
  - AfterSave  ￼

Delete（删除）会触发
  - BeforeDelete
  - AfterDelete  ￼

另外，查询相关还有 AfterFind（不是增删改，但经常一起用）。 ￼
*/

func (c *Channel) BeforeCreate(tx *gorm.DB) error { return nil }
func (c *Channel) AfterCreate(tx *gorm.DB) error  { return nil }

func (c *Channel) BeforeUpdate(tx *gorm.DB) error { return nil }
func (c *Channel) AfterUpdate(tx *gorm.DB) error  { return nil }

func (c *Channel) BeforeDelete(tx *gorm.DB) error { return nil }
func (c *Channel) AfterDelete(tx *gorm.DB) error  { return nil }

func (c *Channel) BeforeSave(tx *gorm.DB) error { return nil } // create & update 都会走
func (c *Channel) AfterSave(tx *gorm.DB) error  { return nil } // create & update 都会走
