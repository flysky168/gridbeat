package auth

import (
	"strings"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// NoAuth disables all auth checks when enabled (dangerous).
// NoAuth 在启用时会关闭所有鉴权（危险模式）。
var NoAuth bool

// ContextUser is stored in Fiber locals.
// ContextUser 存在于 Fiber locals 中。
type ContextUser struct {
	ID       uint
	Username string
	IsRoot   bool
}

// Locals keys / locals key
const (
	LocalUser  = "user"
	LocalToken = "token"
)

// AuthMiddleware verifies JWT and session status.
// AuthMiddleware 验证 JWT 与会话状态。
func AuthMiddleware(db *gorm.DB, jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if NoAuth {
			// Bypass auth / 跳过鉴权
			return c.Next()
		}

		authz := c.Get("Authorization")
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			return response.Unauthorized(c, "missing bearer token")
		}
		tokenStr := strings.TrimSpace(authz[len("Bearer "):])

		claims, err := Parse(jwtSecret, tokenStr)
		if err != nil {
			return response.Unauthorized(c, "invalid token")
		}

		// Server-side session control via auth_tokens table.
		// 通过 auth_tokens 表实现服务端会话控制。
		var t models.AuthToken
		if err := db.Where("jti = ?", claims.ID).First(&t).Error; err != nil {
			return response.Unauthorized(c, "token not found")
		}
		if t.RevokedAt != nil {
			return response.Unauthorized(c, "token revoked")
		}
		if t.ExpiresAt != nil && time.Now().After(*t.ExpiresAt) {
			return response.Unauthorized(c, "token expired")
		}

		// Sliding idle timeout for web sessions.
		// Web 会话滑动空闲超时。
		if t.Type == models.TokenTypeWeb {
			if t.LastSeenAt == nil {
				// If missing, treat as expired to be safe.
				// 若缺失则按过期处理更安全。
				return response.Unauthorized(c, "session invalid")
			}
			idle := time.Duration(t.IdleTimeoutSeconds) * time.Second
			if idle > 0 && time.Since(*t.LastSeenAt) > idle {
				// Mark revoked.
				// 标记撤销。
				now := time.Now()
				_ = db.Model(&t).Updates(map[string]any{"revoked_at": &now}).Error
				return response.Unauthorized(c, "session expired (idle)")
			}
			// Update last seen.
			// 更新最后访问时间。
			now := time.Now()
			_ = db.Model(&t).Updates(map[string]any{"last_seen_at": &now}).Error
		}

		// Put user info into context.
		// 将用户信息放入上下文。
		c.Locals(LocalUser, ContextUser{
			ID:       t.UserID,
			Username: claims.Username,
			IsRoot:   claims.IsRoot,
		})
		c.Locals(LocalToken, t)

		return c.Next()
	}
}

// RequireRoot checks root privilege.
// RequireRoot 检查 root 权限。
func RequireRoot() fiber.Handler {
	return func(c fiber.Ctx) error {
		if NoAuth {
			return c.Next()
		}
		u, ok := c.Locals(LocalUser).(ContextUser)
		if !ok || !u.IsRoot {
			return response.Forbidden(c, "root privilege required")
		}
		return c.Next()
	}
}
