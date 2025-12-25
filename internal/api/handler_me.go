package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/audit"
	"github.com/fluxionwatt/gridbeat/internal/auth"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/fluxionwatt/gridbeat/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ChangePasswordRequest is request body for changing password.
// ChangePasswordRequest 是修改密码的请求体。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangeMyPassword lets a normal user change their own password.
// ChangeMyPassword 允许普通用户自助修改自己的密码。
//
// @Summary Change my password / 修改我的密码
// @Description User changes their own password by providing old_password and new_password.
// @Description 用户提供旧密码与新密码，自助修改密码。
// @Tags me
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ChangePasswordRequest true "request / 请求"
// @Success 200 {object} response.Envelope[any]
// @Failure 400 {object} response.Envelope[any]
// @Failure 401 {object} response.Envelope[any]
// @Router /api/v1/me/password [put]
func (s *Server) ChangeMyPassword(c fiber.Ctx) error {
	u := MustUser(c)
	var req ChangePasswordRequest

	if err := c.Bind().JSON(req); err != nil {
		return response.BadRequest(c, "invalid json")
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		return response.BadRequest(c, "old_password/new_password required")
	}

	var user models.User
	if err := s.DB.First(&user, u.ID).Error; err != nil {
		return response.Internal(c, "db error")
	}

	if err := util.CheckPassword(user.PasswordHash, req.OldPassword); err != nil {
		return response.Unauthorized(c, "old password incorrect")
	}

	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return response.BadRequest(c, "new password invalid")
	}

	if err := s.DB.Model(&user).Update("password_hash", hash).Error; err != nil {
		return response.Internal(c, "update failed")
	}

	// Revoke all tokens of this user after password change for safety.
	// 出于安全考虑，修改密码后撤销该用户所有 token。
	now := time.Now()
	_ = s.DB.Model(&models.AuthToken{}).
		Where("user_id = ? AND revoked_at IS NULL", u.ID).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": u.ID}).Error

	audit.Write(s.DB, c, u, "change_password", "user", nil)
	return response.OK(c, fiber.Map{"changed": true})
}

// CreateMyTokenRequest is request body for creating API token.
// CreateMyTokenRequest 是创建 API Token 的请求体。
type CreateMyTokenRequest struct {
	Name string `json:"name,omitempty"`
}

// TokenInfo describes token metadata.
// TokenInfo 描述 token 元数据。
type TokenInfo struct {
	JTI                string           `json:"jti"`
	Type               models.TokenType `json:"type"`
	Name               string           `json:"name,omitempty"`
	IssuedAt           time.Time        `json:"issued_at"`
	ExpiresAt          *time.Time       `json:"expires_at,omitempty"`
	LastSeenAt         *time.Time       `json:"last_seen_at,omitempty"`
	IdleTimeoutSeconds int              `json:"idle_timeout_seconds"`
	RevokedAt          *time.Time       `json:"revoked_at,omitempty"`
}

// CreateMyAPIToken creates permanent API token for current user.
// CreateMyAPIToken 为当前用户创建永久 API Token。
//
// @Summary Create my API token / 创建我的 API Token
// @Description Create a permanent API token for the current user.
// @Description 为当前用户创建永久 API Token。
// @Tags me
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateMyTokenRequest true "request / 请求"
// @Success 200 {object} response.Envelope[CreateAPITokenResponse]
// @Router /api/v1/me/tokens [post]
func (s *Server) CreateMyAPIToken(c fiber.Ctx) error {
	u := MustUser(c)

	var req CreateMyTokenRequest

	if err := c.Bind().JSON(req); err != nil {
		return err
	}

	jti := uuid.NewString()
	now := time.Now()

	tm := models.AuthToken{
		JTI:       jti,
		UserID:    u.ID,
		Type:      models.TokenTypeAPI,
		Name:      req.Name,
		IssuedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.DB.Create(&tm).Error; err != nil {
		return response.Internal(c, "create token failed")
	}

	jwtStr, err := auth.Sign(s.Cfg.Auth.JWT.Secret, s.Cfg.Auth.JWT.Issuer, jti, u.ID, u.Username, u.IsRoot, string(models.TokenTypeAPI), nil)
	if err != nil {
		return response.Internal(c, "sign token failed")
	}

	audit.Write(s.DB, c, u, "create_token", "token", fiber.Map{"type": "api", "name": req.Name})
	return response.OK(c, CreateAPITokenResponse{Token: jwtStr, JTI: jti, Type: "api", Name: req.Name})
}

// ListMyTokens lists tokens of current user.
// ListMyTokens 列出当前用户的 token。
//
// @Summary List my tokens / 列出我的 Token
// @Description List tokens (web sessions + api tokens) of current user.
// @Description 列出当前用户的 Token（Web 会话 + API Token）。
// @Tags me
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[[]TokenInfo]
// @Router /api/v1/me/tokens [get]
func (s *Server) ListMyTokens(c fiber.Ctx) error {
	u := MustUser(c)
	var tokens []models.AuthToken
	if err := s.DB.Where("user_id = ?", u.ID).Order("created_at desc").Find(&tokens).Error; err != nil {
		return response.Internal(c, "db error")
	}

	out := make([]TokenInfo, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, TokenInfo{
			JTI:                t.JTI,
			Type:               t.Type,
			Name:               t.Name,
			IssuedAt:           t.IssuedAt,
			ExpiresAt:          t.ExpiresAt,
			LastSeenAt:         t.LastSeenAt,
			IdleTimeoutSeconds: t.IdleTimeoutSeconds,
			RevokedAt:          t.RevokedAt,
		})
	}
	return response.OK(c, out)
}

// RevokeMyAllTokens revokes all tokens of current user.
// RevokeMyAllTokens 撤销当前用户的全部 token。
//
// @Summary Revoke all my tokens / 撤销我所有 Token
// @Description Self-service revoke all tokens without admin.
// @Description 普通用户自助撤销自己所有 Token（不走管理员）。
// @Tags me
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[any]
// @Router /api/v1/me/tokens [delete]
func (s *Server) RevokeMyAllTokens(c fiber.Ctx) error {
	u := MustUser(c)
	now := time.Now()
	if err := s.DB.Model(&models.AuthToken{}).
		Where("user_id = ? AND revoked_at IS NULL", u.ID).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": u.ID}).Error; err != nil {
		return response.Internal(c, "revoke failed")
	}
	audit.Write(s.DB, c, u, "revoke_all_tokens", "token", nil)
	return response.OK(c, fiber.Map{"revoked": true})
}

// RevokeMyTokenByJTI revokes one token of current user.
// RevokeMyTokenByJTI 撤销当前用户指定 token。
//
// @Summary Revoke my token / 撤销我的指定 Token
// @Tags me
// @Produce json
// @Security BearerAuth
// @Param jti path string true "token jti / token 标识"
// @Success 200 {object} response.Envelope[any]
// @Failure 404 {object} response.Envelope[any]
// @Router /api/v1/me/tokens/{jti} [delete]
func (s *Server) RevokeMyTokenByJTI(c fiber.Ctx) error {
	u := MustUser(c)
	jti := c.Params("jti")
	if jti == "" {
		return response.BadRequest(c, "jti required")
	}
	now := time.Now()
	res := s.DB.Model(&models.AuthToken{}).
		Where("user_id = ? AND jti = ? AND revoked_at IS NULL", u.ID, jti).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": u.ID})
	if res.Error != nil {
		return response.Internal(c, "db error")
	}
	if res.RowsAffected == 0 {
		return response.NotFound(c, "token not found")
	}
	audit.Write(s.DB, c, u, "revoke_token", "token", fiber.Map{"jti": jti})
	return response.OK(c, fiber.Map{"revoked": true})
}

// ListMySessions lists active web sessions for current user.
// ListMySessions 列出当前用户 Web 会话列表。
//
// @Summary List my web sessions / 列出我的 Web 会话
// @Description List active web sessions (type=web, not revoked).
// @Description 列出有效 Web 会话（type=web 且未撤销）。
// @Tags me
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[[]TokenInfo]
// @Router /api/v1/me/sessions [get]
func (s *Server) ListMySessions(c fiber.Ctx) error {
	u := MustUser(c)
	var sessions []models.AuthToken
	if err := s.DB.Where("user_id = ? AND type = ?", u.ID, models.TokenTypeWeb).
		Order("last_seen_at desc").
		Find(&sessions).Error; err != nil {
		return response.Internal(c, "db error")
	}
	out := make([]TokenInfo, 0, len(sessions))
	for _, t := range sessions {
		out = append(out, TokenInfo{
			JTI:                t.JTI,
			Type:               t.Type,
			Name:               t.Name,
			IssuedAt:           t.IssuedAt,
			ExpiresAt:          t.ExpiresAt,
			LastSeenAt:         t.LastSeenAt,
			IdleTimeoutSeconds: t.IdleTimeoutSeconds,
			RevokedAt:          t.RevokedAt,
		})
	}
	return response.OK(c, out)
}

// KickMySession revokes a web session by jti.
// KickMySession 按 jti 踢掉指定 Web 会话。
//
// @Summary Kick my session / 踢掉我的会话
// @Tags me
// @Produce json
// @Security BearerAuth
// @Param jti path string true "session jti / 会话标识"
// @Success 200 {object} response.Envelope[any]
// @Failure 404 {object} response.Envelope[any]
// @Router /api/v1/me/sessions/{jti} [delete]
func (s *Server) KickMySession(c fiber.Ctx) error {
	u := MustUser(c)
	jti := c.Params("jti")
	if jti == "" {
		return response.BadRequest(c, "jti required")
	}
	now := time.Now()
	res := s.DB.Model(&models.AuthToken{}).
		Where("user_id = ? AND jti = ? AND type = ? AND revoked_at IS NULL", u.ID, jti, models.TokenTypeWeb).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": u.ID})
	if res.Error != nil {
		return response.Internal(c, "db error")
	}
	if res.RowsAffected == 0 {
		return response.NotFound(c, "session not found")
	}
	audit.Write(s.DB, c, u, "kick_session", "session", fiber.Map{"jti": jti})
	return response.OK(c, fiber.Map{"kicked": true})
}

// AuditLogItem is response item for audit log query.
// AuditLogItem 是审计日志查询的返回项。
type AuditLogItem struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Detail    string    `json:"detail,omitempty"`
	IP        string    `json:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLogPage is paged response.
// AuditLogPage 是分页响应。
type AuditLogPage struct {
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
	Items []AuditLogItem `json:"items"`
}

// ListMyAuditLogs lists audit logs for current user.
// ListMyAuditLogs 列出当前用户的审计日志。
//
// @Summary List my audit logs / 查看我的审计日志
// @Description Normal user can only view their own audit logs.
// @Description 普通用户只能查看自己的审计日志。
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param page query int false "page / 页码" default(1)
// @Param page_size query int false "page size / 每页数量" default(20)
// @Param from query string false "start time (RFC3339 or YYYY-MM-DD) / 开始时间"
// @Param to query string false "end time (RFC3339 or YYYY-MM-DD) / 结束时间"
// @Success 200 {object} response.Envelope[AuditLogPage]
// @Router /api/v1/me/audit/logs [get]
func (s *Server) ListMyAuditLogs(c fiber.Ctx) error {
	u := MustUser(c)
	return s.listAuditLogs(c, &u, false)
}

// parseTimeQuery parses "from/to" query.
// parseTimeQuery 解析 from/to 查询参数。
func parseTimeQuery(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	// Try RFC3339 first / 优先尝试 RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return &t, nil
	}
	// Then date-only / 再尝试仅日期
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t, nil
	}
	return nil, errors.New("invalid time")
}

// listAuditLogs is shared implementation for me/admin.
// listAuditLogs 是 me/admin 共享实现。
func (s *Server) listAuditLogs(c fiber.Ctx, user *auth.ContextUser, isAdmin bool) error {
	pq := util.ParsePageQuery(c)
	offset, limit := pq.OffsetLimit()

	from, err := parseTimeQuery(c.Query("from"))
	if err != nil {
		return response.BadRequest(c, "invalid from")
	}
	to, err := parseTimeQuery(c.Query("to"))
	if err != nil {
		return response.BadRequest(c, "invalid to")
	}

	q := s.DB.Model(&models.AuditLog{})
	if !isAdmin && user != nil {
		q = q.Where("user_id = ?", user.ID)
	} else if isAdmin {
		// Optional filter by user_id
		// 可选按 user_id 过滤
		if uidStr := c.Query("user_id"); uidStr != "" {
			if uid, err := strconv.ParseUint(uidStr, 10, 64); err == nil {
				q = q.Where("user_id = ?", uint(uid))
			}
		}
	}
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return response.Internal(c, "db error")
	}

	var logs []models.AuditLog
	if err := q.Order("created_at desc").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return response.Internal(c, "db error")
	}

	items := make([]AuditLogItem, 0, len(logs))
	for _, l := range logs {
		items = append(items, AuditLogItem{
			ID:        l.ID,
			UserID:    l.UserID,
			Username:  l.Username,
			Action:    l.Action,
			Resource:  l.Resource,
			Method:    l.Method,
			Path:      l.Path,
			Detail:    l.Detail,
			IP:        l.IP,
			UserAgent: l.UserAgent,
			CreatedAt: l.CreatedAt,
		})
	}

	return response.OK(c, AuditLogPage{
		Total: total,
		Page:  pq.Page,
		Size:  pq.PageSize,
		Items: items,
	})
}
