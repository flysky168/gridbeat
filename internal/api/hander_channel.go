package api

import (
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/gofiber/fiber/v3"
)

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
func (s *Server) ListOnlineChanel(c fiber.Ctx) error {

	var cs []models.Channel
	if err := s.DB.Order("device asc").Find(&cs).Error; err != nil {
		return response.Internal(c, "db error")
	}

	for i, _ := range cs {
		in, ok := s.Mgr.Get("mbus", cs[i].UUID)
		if ok {
			status := in.Get().(models.ChannelStatus)
			cs[i].Status = status
		}
	}
	return response.OK(c, cs)
}
