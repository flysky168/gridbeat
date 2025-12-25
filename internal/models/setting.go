package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScalarJSON stores any JSON value (scalar/object/array) safely.
// ScalarJSON 用于安全存放任意 JSON 值（标量/对象/数组），避免 JSONB 标量 Scan 失败。
type ScalarJSON []byte

// Scan implements sql.Scanner / 实现 sql.Scanner
func (j *ScalarJSON) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		// Postgres JSON/JSONB often returns []byte / Postgres JSON/JSONB 通常返回 []byte
		*j = append((*j)[:0], v...)
		return nil
	case string:
		// Some drivers return string / 部分驱动返回 string
		*j = append((*j)[:0], []byte(v)...)
		return nil
	case int64, float64, bool:
		// Some drivers may decode JSONB scalar into native types / 某些驱动可能把 JSONB 标量解成原生类型
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal json scalar failed / 序列化 JSON 标量失败: %w", err)
		}
		*j = b
		return nil
	default:
		return fmt.Errorf("unsupported Scan type %T for ScalarJSON / ScalarJSON 不支持的 Scan 类型: %T", src, src)
	}
}

// Value implements driver.Valuer / 实现 driver.Valuer
func (j ScalarJSON) Value() (driver.Value, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// Setting 系统配置表
// Setting represents system settings table.
//
// value_json 存放 JSON 原始值：可以是 number/string/bool
// value_type 作为判别字段：int/string/bool
type Setting struct {
	// UUID string primary key
	// UUID 字符串主键
	ID string `json:"id" gorm:"primaryKey;type:char(36)"`

	// Unique key name
	// 唯一配置名
	Name string `json:"name" gorm:"size:128;uniqueIndex;not null"`

	// Discriminator for value type: int|string|bool
	// 值类型判别字段：int|string|bool
	ValueType string `json:"value_type" gorm:"size:16;not null;index"`

	// ValueJSON stores raw JSON scalar (10/"abc"/true) / 原始 JSON 标量
	ValueJSON ScalarJSON `json:"value_json" gorm:"type:jsonb;not null" swaggertype:"string" example:"10"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名 / TableName specifies table name.
func (Setting) TableName() string { return "setting" }

// BeforeCreate 在创建前生成 UUID / generate UUID before insert.
func (s *Setting) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}
