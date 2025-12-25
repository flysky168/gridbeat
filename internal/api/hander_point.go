package api

import (
	"strings"

	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// CreateSettingRequest 创建 setting 请求
// CreateSettingRequest is request body for create.
type CreatePointRequest struct {
	TypeKey string                      `json:"type_key" example:"sampling_interval_sec"`
	Model   string                      `json:"model" example:"model"` // int|string|bool
	Point   []models.DeveltypePointBase `json:"Point"`                 // 10 / "abc" / true
}

// CreatePoint
// @Summary Create point
// @Tags Point
// @Accept json
// @Produce json
// @Param body body CreatePointRequest true "create point"
// @Success 201 {object} models.point
// @Failure 400 {object} APIError
// @Failure 409 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/point [post]
func (h *Server) CreatePoint(c fiber.Ctx) error {
	var req CreatePointRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(APIError{Message: err.Error()})
	}

	req.TypeKey = strings.TrimSpace(req.TypeKey)

	var cnt int64
	if err := h.DB.Model(&models.DeviceTypePoint{}).Where("type_key = ?", req.TypeKey).Count(&cnt).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}
	if cnt > 0 {
		return c.Status(409).JSON(APIError{Message: "TypeKey already exists"})
	}

	items := make([]models.DeviceTypePoint, 0, len(req.Point))

	for i := range req.Point {
		items[i].ID = uuid.NewString()
		items[i].TypeKey = req.TypeKey
		items[i].Model = req.Model
	}

	if err := h.DB.CreateInBatches(items, len(req.Point)).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}

	return c.Status(201).JSON(items)
}
