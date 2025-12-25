package models

import (
	"time"

	"gorm.io/gorm"
)

type Site struct {
	gorm.Model
	UUID        string    `gorm:"primaryKey;column:uuid;size:36;uniqueIndex;not null" json:"uuid"`
	Name        string    `gorm:"column:name;size:1024;uniqueIndex;not null" json:"name"`
	Power       uint64    // 电站额定充放电功率，单位 瓦特
	Store       uint64    // 电站额定储能容量，单位 瓦特H
	GridConTime time.Time // 并网/启用时间
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 用来显式指定表名（可选）
func (Site) TableName() string {
	return "site"
}

// Array  子阵
type Array struct {
	SiteID    string
	Site      Site      `gorm:"references:UUID"`
	UUID      string    `gorm:"primaryKey;column:uuid;size:36;uniqueIndex;not null" json:"uuid"`
	Name      string    `gorm:"column:name;size:150;uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 用来显式指定表名（可选）
func (Array) TableName() string {
	return "array"
}
