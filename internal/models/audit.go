package models

import "time"

// AuditLog stores immutable audit records (no delete API).
// AuditLog 保存不可变审计记录（不提供删除 API）。
type AuditLog struct {
	ID uint `gorm:"primaryKey" json:"id"`

	UserID   uint   `gorm:"index;not null" json:"user_id"`
	Username string `gorm:"index;size:64;not null" json:"username"`

	// Action: "login", "create_user", "change_password", ...
	// Action：动作，如 "login", "create_user", "change_password" 等。
	Action string `gorm:"size:64;index;not null" json:"action"`

	// Resource: "user", "token", "session", "audit"
	// Resource：资源类型，如 "user", "token", "session", "audit"
	Resource string `gorm:"size:64;index;not null" json:"resource"`

	Method string `gorm:"size:16;not null" json:"method"`
	Path   string `gorm:"size:256;not null" json:"path"`

	// Detail is extra JSON/text.
	// Detail 是额外信息（JSON/文本）。
	Detail string `gorm:"type:text" json:"detail,omitempty"`

	IP        string `gorm:"size:64" json:"ip,omitempty"`
	UserAgent string `gorm:"size:256" json:"user_agent,omitempty"`

	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

// TableName 用来显式指定表名（可选）
func (AuditLog) TableName() string {
	return "log"
}
