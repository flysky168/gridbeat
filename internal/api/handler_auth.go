package api

import (
	"errors"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/audit"
	"github.com/fluxionwatt/gridbeat/internal/auth"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/fluxionwatt/gridbeat/internal/util"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginWebRequest is request body for web login.
// LoginWebRequest 是 Web 登录的请求体。
type LoginWebRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginWebResponse is response for web login.
// LoginWebResponse 是 Web 登录的响应。
type LoginWebResponse struct {
	Token              string `json:"token"`
	JTI                string `json:"jti"`
	IdleTimeoutSeconds int    `json:"idle_timeout_seconds"`
	Type               string `json:"type"`
}

// LoginWeb handles web login and returns a "web" session token.
//
// @Summary Web login / Web 登录
// @Description Create a web session token with sliding idle timeout.
// @Description 创建带滑动空闲超时的 Web 会话 Token。
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginWebRequest true "login request / 登录请求"
// @Success 200 {object} response.Envelope[LoginWebResponse]
// @Failure 400 {object} response.Envelope[any]
// @Failure 401 {object} response.Envelope[any]
// @Router /api/v1/auth/login [post]
func (s *Server) LoginWeb(c fiber.Ctx) error {

	req := new(LoginWebRequest)
	if err := c.Bind().JSON(req); err != nil {
		return response.BadRequest(c, "invalid json")
	}
	if req.Username == "" || req.Password == "" {
		return response.BadRequest(c, "username/password required")
	}

	var u models.User
	if err := s.DB.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Unauthorized(c, "invalid credentials")
		}
		return response.Internal(c, "db error")
	}

	if err := util.CheckPassword(u.PasswordHash, req.Password); err != nil {
		return response.Unauthorized(c, "invalid credentials")
	}

	jti := uuid.NewString()
	now := time.Now()
	idle := int(s.Cfg.WebIdleTimeout().Seconds())

	// Persist session metadata.
	// 持久化会话元数据。
	tm := models.AuthToken{
		JTI:                jti,
		UserID:             u.ID,
		Type:               models.TokenTypeWeb,
		IssuedAt:           now,
		LastSeenAt:         &now,
		IdleTimeoutSeconds: idle,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.DB.Create(&tm).Error; err != nil {
		return response.Internal(c, "create session failed")
	}

	jwtStr, err := auth.Sign(s.Cfg.Auth.JWT.Secret, s.Cfg.Auth.JWT.Issuer, jti, u.ID, u.Username, u.IsRoot, string(models.TokenTypeWeb), nil)
	if err != nil {
		return response.Internal(c, "sign token failed")
	}

	audit.Write(s.DB, c, auth.ContextUser{ID: u.ID, Username: u.Username, IsRoot: u.IsRoot}, "login", "auth", fiber.Map{"type": "web"})

	/*
			data := fiber.Map{
			"token": "9944b09199c62bcf9418ad846dd0e4bbdfc6ee4b",
			"user": fiber.Map{
				"id":       "84a35e05-531f-4d96-8d5b-bc8a7a358493",
				"username": "root",
			},
		}
	*/

	return response.OK(c, LoginWebResponse{
		Token:              jwtStr,
		JTI:                jti,
		IdleTimeoutSeconds: idle,
		Type:               "web",
	})
}

// CreateAPITokenRequest is request body for creating a permanent API token.
// CreateAPITokenRequest 是创建永久 API Token 的请求体。
type CreateAPITokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// CreateAPITokenResponse is response for API token creation.
// CreateAPITokenResponse 是 API Token 创建响应。
type CreateAPITokenResponse struct {
	Token string `json:"token"`
	JTI   string `json:"jti"`
	Type  string `json:"type"`
	Name  string `json:"name,omitempty"`
}

// CreateAPITokenByPassword creates a permanent API token by username+password.
//
// @Summary Create API token by password / 通过密码创建 API Token
// @Description Create a permanent API token for subsequent authenticated API calls.
// @Description 通过用户名密码创建永久 API Token，用于后续需要鉴权的 API。
// @Tags auth
// @Accept json
// @Produce json
// @Param body body CreateAPITokenRequest true "request / 请求"
// @Success 200 {object} response.Envelope[CreateAPITokenResponse]
// @Failure 400 {object} response.Envelope[any]
// @Failure 401 {object} response.Envelope[any]
// @Router /api/v1/auth/token [post]
func (s *Server) CreateAPITokenByPassword(c fiber.Ctx) error {
	var req CreateAPITokenRequest

	if err := c.Bind().JSON(req); err != nil {
		return response.BadRequest(c, "invalid json")
	}
	if req.Username == "" || req.Password == "" {
		return response.BadRequest(c, "username/password required")
	}

	var u models.User
	if err := s.DB.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.Unauthorized(c, "invalid credentials")
		}
		return response.Internal(c, "db error")
	}

	if err := util.CheckPassword(u.PasswordHash, req.Password); err != nil {
		return response.Unauthorized(c, "invalid credentials")
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

	audit.Write(s.DB, c, auth.ContextUser{ID: u.ID, Username: u.Username, IsRoot: u.IsRoot}, "create_token", "token", fiber.Map{"type": "api", "name": req.Name})

	return response.OK(c, CreateAPITokenResponse{
		Token: jwtStr,
		JTI:   jti,
		Type:  "api",
		Name:  req.Name,
	})
}

// Logout revokes current token.
//
// @Summary Logout / 注销
// @Description Revoke current token (web session or api token).
// @Description 撤销当前 Token（Web 会话或 API Token）。
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[any]
// @Failure 401 {object} response.Envelope[any]
// @Router /api/v1/auth/logout [post]
func (s *Server) Logout(c fiber.Ctx) error {
	u := MustUser(c)
	t, ok := c.Locals(auth.LocalToken).(models.AuthToken)
	if !ok {
		return response.Unauthorized(c, "token missing")
	}

	now := time.Now()
	if err := s.DB.Model(&models.AuthToken{}).
		Where("jti = ? AND revoked_at IS NULL", t.JTI).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": u.ID}).
		Error; err != nil {
		return response.Internal(c, "revoke failed")
	}

	audit.Write(s.DB, c, u, "logout", "auth", fiber.Map{"jti": t.JTI, "type": t.Type})
	return response.OK(c, fiber.Map{"revoked": true})
}
