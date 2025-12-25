package api

import (
	"time"

	"github.com/fluxionwatt/gridbeat/internal/audit"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
)

// SessionItem is web session info.
// SessionItem 是 Web 会话信息。
type SessionItem struct {
	JTI                string     `json:"jti"`
	UserID             uint       `json:"user_id"`
	Username           string     `json:"username"`
	IssuedAt           time.Time  `json:"issued_at"`
	LastSeenAt         *time.Time `json:"last_seen_at,omitempty"`
	IdleTimeoutSeconds int        `json:"idle_timeout_seconds"`
	RevokedAt          *time.Time `json:"revoked_at,omitempty"`
}

// AdminListSessions lists all web sessions (root only).
// AdminListSessions 列出所有 Web 会话（仅 root）。
//
// @Summary List all web sessions / 列出所有 Web 会话
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[[]SessionItem]
// @Router /api/v1/admin/sessions [get]
func (s *Server) AdminListSessions(c fiber.Ctx) error {
	type row struct {
		models.AuthToken
		Username string
	}
	var rows []row
	if err := s.DB.Table("auth_tokens").
		Select("auth_tokens.*, users.username as username").
		Joins("left join users on users.id = auth_tokens.user_id").
		Where("auth_tokens.type = ?", models.TokenTypeWeb).
		Order("auth_tokens.last_seen_at desc").
		Scan(&rows).Error; err != nil {
		return response.Internal(c, "db error")
	}

	out := make([]SessionItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, SessionItem{
			JTI:                r.JTI,
			UserID:             r.UserID,
			Username:           r.Username,
			IssuedAt:           r.IssuedAt,
			LastSeenAt:         r.LastSeenAt,
			IdleTimeoutSeconds: r.IdleTimeoutSeconds,
			RevokedAt:          r.RevokedAt,
		})
	}
	return response.OK(c, out)
}

// AdminKickSession revokes any web session by jti (root only).
// AdminKickSession root 按 jti 撤销任意 Web 会话。
//
// @Summary Kick a web session / 踢掉 Web 会话
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param jti path string true "session jti / 会话标识"
// @Success 200 {object} response.Envelope[any]
// @Router /api/v1/admin/sessions/{jti} [delete]
func (s *Server) AdminKickSession(c fiber.Ctx) error {
	admin := MustUser(c)
	jti := c.Params("jti")
	if jti == "" {
		return response.BadRequest(c, "jti required")
	}

	now := time.Now()
	res := s.DB.Model(&models.AuthToken{}).
		Where("jti = ? AND type = ? AND revoked_at IS NULL", jti, models.TokenTypeWeb).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": admin.ID})
	if res.Error != nil {
		return response.Internal(c, "db error")
	}
	if res.RowsAffected == 0 {
		return response.NotFound(c, "session not found")
	}

	audit.Write(s.DB, c, admin, "kick_session", "session", fiber.Map{"jti": jti})
	return response.OK(c, fiber.Map{"kicked": true})
}
