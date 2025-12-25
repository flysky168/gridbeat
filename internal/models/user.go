package models

import "time"

// User represents a local account.
// User 表示本地账号。
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	IsRoot       bool       `gorm:"not null;default:false" json:"is_root"`
	LastLogin    *time.Time `json:"last_login"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 用来显式指定表名（可选）
func (User) TableName() string {
	return "user"
}
