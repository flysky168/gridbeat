package models

import "time"

// TokenType defines token categories.
// TokenType 定义 token 分类。
type TokenType string

const (
	// TokenTypeWeb is for web login sessions (idle-timeout).
	// TokenTypeWeb 用于 Web 登录会话（空闲超时）。
	TokenTypeWeb TokenType = "web"

	// TokenTypeAPI is for permanent API token (PAT-like).
	// TokenTypeAPI 用于永久 API Token（类似 PAT）。
	TokenTypeAPI TokenType = "api"
)

// AuthToken stores token metadata (server-side session control).
// AuthToken 存储 token 元数据（服务端可控会话）。
type AuthToken struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// JTI is the unique identifier of JWT.
	// JTI 是 JWT 的唯一标识。
	JTI string `gorm:"uniqueIndex;size:64;not null" json:"jti"`

	UserID uint `gorm:"index;not null" json:"user_id"`
	User   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Type TokenType `gorm:"size:16;index;not null" json:"type"`

	// Name is optional token label.
	// Name 是可选的 token 备注。
	Name string `gorm:"size:128" json:"name,omitempty"`

	IssuedAt time.Time `gorm:"not null" json:"issued_at"`

	// ExpiresAt is optional absolute expiration.
	// ExpiresAt 是可选的绝对过期时间。
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// LastSeenAt enables sliding idle timeout for web sessions.
	// LastSeenAt 用于 Web 会话的滑动空闲超时。
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`

	// IdleTimeoutSeconds is used only for web sessions.
	// IdleTimeoutSeconds 仅用于 Web 会话。
	IdleTimeoutSeconds int `gorm:"not null;default:0" json:"idle_timeout_seconds"`

	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	RevokedBy *uint      `json:"revoked_by,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
