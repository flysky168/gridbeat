package api

import (
	"time"

	"github.com/fluxionwatt/gridbeat/internal/audit"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
)

// AdminTokenItem is token info for admin.
// AdminTokenItem 是管理员查看的 token 信息。
type AdminTokenItem struct {
	JTI                string           `json:"jti"`
	UserID             uint             `json:"user_id"`
	Username           string           `json:"username"`
	Type               models.TokenType `json:"type"`
	Name               string           `json:"name,omitempty"`
	IssuedAt           time.Time        `json:"issued_at"`
	ExpiresAt          *time.Time       `json:"expires_at,omitempty"`
	LastSeenAt         *time.Time       `json:"last_seen_at,omitempty"`
	IdleTimeoutSeconds int              `json:"idle_timeout_seconds"`
	RevokedAt          *time.Time       `json:"revoked_at,omitempty"`
	RevokedBy          *uint            `json:"revoked_by,omitempty"`
}

// AdminListTokens lists all tokens (root only).
// AdminListTokens root 列出所有 token。
//
// @Summary List all tokens / 列出所有 Token
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[[]AdminTokenItem]
// @Router /api/v1/admin/tokens [get]
func (s *Server) AdminListTokens(c fiber.Ctx) error {
	type row struct {
		models.AuthToken
		Username string
	}
	var rows []row
	if err := s.DB.Table("auth_tokens").
		Select("auth_tokens.*, users.username as username").
		Joins("left join users on users.id = auth_tokens.user_id").
		Order("auth_tokens.created_at desc").
		Scan(&rows).Error; err != nil {
		return response.Internal(c, "db error")
	}

	out := make([]AdminTokenItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, AdminTokenItem{
			JTI:                r.JTI,
			UserID:             r.UserID,
			Username:           r.Username,
			Type:               r.Type,
			Name:               r.Name,
			IssuedAt:           r.IssuedAt,
			ExpiresAt:          r.ExpiresAt,
			LastSeenAt:         r.LastSeenAt,
			IdleTimeoutSeconds: r.IdleTimeoutSeconds,
			RevokedAt:          r.RevokedAt,
			RevokedBy:          r.RevokedBy,
		})
	}
	return response.OK(c, out)
}

// AdminRevokeToken revokes any token by jti (root only).
// AdminRevokeToken root 按 jti 撤销任意 token。
//
// @Summary Revoke token / 撤销 Token
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param jti path string true "token jti / token 标识"
// @Success 200 {object} response.Envelope[any]
// @Router /api/v1/admin/tokens/{jti} [delete]
func (s *Server) AdminRevokeToken(c fiber.Ctx) error {
	admin := MustUser(c)
	jti := c.Params("jti")
	if jti == "" {
		return response.BadRequest(c, "jti required")
	}

	now := time.Now()
	res := s.DB.Model(&models.AuthToken{}).
		Where("jti = ? AND revoked_at IS NULL", jti).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": admin.ID})
	if res.Error != nil {
		return response.Internal(c, "db error")
	}
	if res.RowsAffected == 0 {
		return response.NotFound(c, "token not found")
	}

	audit.Write(s.DB, c, admin, "revoke_token", "token", fiber.Map{"jti": jti})
	return response.OK(c, fiber.Map{"revoked": true})
}
