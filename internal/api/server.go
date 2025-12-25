package api

import (
	"time"

	"github.com/fluxionwatt/gridbeat/core"
	_ "github.com/fluxionwatt/gridbeat/docs"
	"github.com/fluxionwatt/gridbeat/internal/auth"
	"github.com/fluxionwatt/gridbeat/internal/config"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
	mqtt "github.com/mochi-mqtt/server/v2"
	"gorm.io/gorm"
)

// Server holds dependencies for HTTP handlers.
// Server 保存 HTTP handler 依赖项。
type Server struct {
	DB   *gorm.DB
	Cfg  *config.Config
	MQTT *mqtt.Server
	Mgr  *core.InstanceManager
}

// New creates server instance.
// New 创建 Server 实例。
//func New(db *gorm.DB, cfg *config.Config) *Server {
//	return &Server{DB: db, Cfg: cfg}
//}

// App builds Fiber app with routes.
// App 构建包含路由的 Fiber 应用。
func (s *Server) Route(app *fiber.App) *fiber.App {

	v1 := app.Group("/api/v1")

	// Public / 无需鉴权 API
	v1.Get("/health", func(c fiber.Ctx) error {
		return response.OK(c, fiber.Map{"status": "ok"})
	})
	v1.Get("/public/ping", func(c fiber.Ctx) error {
		return response.OK(c, fiber.Map{"pong": true})
	})

	// Auth / 鉴权
	v1.Post("/auth/login", s.LoginWeb)
	v1.Post("/auth/token", s.CreateAPITokenByPassword)

	// Protected group / 需要鉴权的 API
	protected := v1.Group("", auth.AuthMiddleware(s.DB, s.Cfg.Auth.JWT.Secret))
	protected.Post("/auth/logout", s.Logout)

	protected.Get("/system/version", s.Version)
	protected.Get("/maintenance/overview", s.MaintenanceOverview)

	// Self-service / 自助 API
	me := protected.Group("/me")
	me.Put("/password", s.ChangeMyPassword)
	me.Post("/tokens", s.CreateMyAPIToken)
	me.Get("/tokens", s.ListMyTokens)
	me.Delete("/tokens", s.RevokeMyAllTokens)
	me.Delete("/tokens/:jti", s.RevokeMyTokenByJTI)

	me.Get("/sessions", s.ListMySessions)
	me.Delete("/sessions/:jti", s.KickMySession)

	me.Get("/audit/logs", s.ListMyAuditLogs)

	// Admin / 管理 API（root only）
	admin := protected.Group("/admin", auth.RequireRoot())
	admin.Get("/users", s.AdminListUsers)
	admin.Post("/users", s.AdminCreateUser)
	admin.Put("/users/:id/password", s.AdminResetUserPassword)
	admin.Delete("/users/:id", s.AdminDeleteUser)

	admin.Get("/sessions", s.AdminListSessions)
	admin.Delete("/sessions/:jti", s.AdminKickSession)

	admin.Get("/tokens", s.AdminListTokens)
	admin.Delete("/tokens/:jti", s.AdminRevokeToken)

	admin.Get("/audit/logs", s.AdminListAuditLogs)

	serial := v1.Group("/serial", auth.AuthMiddleware(s.DB, s.Cfg.Auth.JWT.Secret))
	registerSerialRoutes(serial, s.DB)

	channels := v1.Group("/channels", auth.AuthMiddleware(s.DB, s.Cfg.Auth.JWT.Secret))
	channels.Get("/", s.ListOnlineChanel)

	point := v1.Group("/point", auth.AuthMiddleware(s.DB, s.Cfg.Auth.JWT.Secret))
	// 创建 / Create
	// POST /api/v1/point
	point.Post("/", s.CreatePoint)

	// settings
	settings := v1.Group("/settings", auth.AuthMiddleware(s.DB, s.Cfg.Auth.JWT.Secret))

	// 列表 / List
	// GET /api/v1/settings
	settings.Get("/", s.ListSettings)

	// 创建 / Create
	// POST /api/v1/settings
	settings.Post("/", s.CreateSetting)

	// 按 ID 更新（支持 value_type + value 的 PATCH）
	// Update by ID (PATCH value_type + value)
	// PATCH /api/v1/settings/:id
	settings.Patch("/:id", s.UpdateSettingByID)

	// ------- 下面是“可选”路由（如果你已有对应 handler，就取消注释） -------
	// The following routes are optional (uncomment if handlers exist).

	// // 按 ID 获取 / Get by ID
	// // GET /api/v1/settings/:id
	settings.Get("/:id", s.GetSettingByID)

	// // 按 name 获取 / Get by name
	// // GET /api/v1/settings/by-name/:name
	settings.Get("/by-name/:name", s.GetSettingByName)

	// // 按 name upsert value / Upsert value by name
	// // PUT /api/v1/settings/by-name/:name
	settings.Put("/by-name/:name", s.UpsertSettingValueByName)

	// // 按 ID 删除 / Delete by ID
	// // DELETE /api/v1/settings/:id
	settings.Delete("/:id", s.DeleteSettingByID)

	return app
}

// StartAuditRetentionJob starts periodic cleanup according to retention policy.
// StartAuditRetentionJob 根据保留策略启动周期性清理任务。
func (s *Server) StartAuditRetentionJob(stop <-chan struct{}) {
	days := s.Cfg.Audit.RetentionDays
	if days <= 0 {
		days = 120
	}

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -days)
				// Auto-clean old audit logs (system only, no API for deleting).
				// 自动清理旧审计日志（系统行为，不提供删除 API）。
				_ = s.DB.Where("created_at < ?", cutoff).Delete(&models.AuditLog{}).Error
			}
		}
	}()
}

// MustUser reads authenticated user from context locals.
// MustUser 从上下文 locals 读取鉴权用户信息。
func MustUser(c fiber.Ctx) auth.ContextUser {
	if auth.NoAuth {
		// In no-auth mode, treat as root for convenience.
		// 在 no-auth 模式下，默认按 root 处理方便调试。
		return auth.ContextUser{ID: 0, Username: "noauth", IsRoot: true}
	}
	u, _ := c.Locals(auth.LocalUser).(auth.ContextUser)
	return u
}
