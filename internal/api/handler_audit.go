package api

import (
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
)

// AdminListAuditLogs lists audit logs for all users (root only).
// AdminListAuditLogs root 查询所有用户审计日志。
//
// @Summary List audit logs (admin) / 查询审计日志（管理员）
// @Description Root can query audit logs of all users with pagination and time range filter.
// @Description root 用户可分页查询全部审计日志，并支持时间范围过滤。
// @Tags audit
// @Produce json
// @Security BearerAuth
// @Param page query int false "page / 页码" default(1)
// @Param page_size query int false "page size / 每页数量" default(20)
// @Param from query string false "start time (RFC3339 or YYYY-MM-DD) / 开始时间"
// @Param to query string false "end time (RFC3339 or YYYY-MM-DD) / 结束时间"
// @Param user_id query int false "filter by user_id / 按用户ID过滤（可选）"
// @Success 200 {object} response.Envelope[AuditLogPage]
// @Router /api/v1/admin/audit/logs [get]
func (s *Server) AdminListAuditLogs(c fiber.Ctx) error {
	return s.listAuditLogs(c, nil, true)
}

// swagger imports note:
// We intentionally keep swagger annotations in handlers.
// 我们在 handler 中保留 swagger 注释。
// If you use swaggo/swag, run:
// 如果你使用 swaggo/swag，请运行：
//   swag init -g cmd/gridbeat/main.go -o docs
var _ = response.CodeOK
