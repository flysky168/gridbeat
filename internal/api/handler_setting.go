package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type APIError struct {
	Message string `json:"message"`
}

// UpsertSettingByNameRequest 按 name 写入/更新（value 支持 int/string/bool）
// UpsertSettingByNameRequest upserts a setting by name (value supports int/string/bool).
type UpsertSettingByNameRequest struct {
	ValueType string          `json:"value_type" example:"bool"` // int|string|bool
	Value     json.RawMessage `json:"value" example:"true"`      // 10 / "abc" / true
}

// CreateSettingRequest 创建 setting 请求
// CreateSettingRequest is request body for create.
type CreateSettingRequest struct {
	Name      string          `json:"name" example:"sampling_interval_sec"`
	ValueType string          `json:"value_type" example:"int"` // int|string|bool
	Value     json.RawMessage `json:"value" example:"10"`       // 10 / "abc" / true
}

// UpdateSettingRequest 更新 setting 请求
// UpdateSettingRequest is request body for update.
type UpdateSettingRequest struct {
	ValueType *string         `json:"value_type" example:"bool"`
	Value     json.RawMessage `json:"value" example:"true"`
}

// validateValueTypeAndJSON validates that value matches value_type.
// 校验：value_json 的 JSON 类型必须匹配 value_type
func validateValueTypeAndJSON(valueType string, raw json.RawMessage) error {
	valueType = strings.TrimSpace(strings.ToLower(valueType))
	if valueType != "int" && valueType != "string" && valueType != "bool" {
		return errors.New("value_type must be one of: int|string|bool")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return errors.New("value is required")
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return errors.New("value must be valid json")
	}

	switch valueType {
	case "string":
		if _, ok := v.(string); !ok {
			return errors.New("value must be a JSON string when value_type=string")
		}
	case "bool":
		if _, ok := v.(bool); !ok {
			return errors.New("value must be a JSON boolean when value_type=bool")
		}
	case "int":
		// JSON numbers decode as json.Number due to UseNumber()
		num, ok := v.(json.Number)
		if !ok {
			return errors.New("value must be a JSON number when value_type=int")
		}
		// Ensure it's an integer (no decimals)
		if _, err := num.Int64(); err != nil {
			return errors.New("value must be an integer when value_type=int")
		}
	}
	return nil
}

// ListSettings
// @Summary List settings
// @Description List all settings
// @Tags Setting
// @Produce json
// @Success 200 {array} models.Setting
// @Failure 500 {object} APIError
// @Router /api/v1/settings [get]
func (h *Server) ListSettings(c fiber.Ctx) error {
	var items []models.Setting
	if err := h.DB.Order("name asc").Find(&items).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}
	return c.JSON(items)
}

// CreateSetting
// @Summary Create setting
// @Tags Setting
// @Accept json
// @Produce json
// @Param body body CreateSettingRequest true "create setting"
// @Success 201 {object} models.Setting
// @Failure 400 {object} APIError
// @Failure 409 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings [post]
func (h *Server) CreateSetting(c fiber.Ctx) error {
	var req CreateSettingRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(APIError{Message: err.Error()})
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ValueType = strings.TrimSpace(req.ValueType)

	if req.Name == "" {
		return c.Status(400).JSON(APIError{Message: "name is required"})
	}
	if err := validateValueTypeAndJSON(req.ValueType, req.Value); err != nil {
		return c.Status(400).JSON(APIError{Message: err.Error()})
	}

	var cnt int64
	if err := h.DB.Model(&models.Setting{}).Where("name = ?", req.Name).Count(&cnt).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}
	if cnt > 0 {
		return c.Status(409).JSON(APIError{Message: "setting name already exists"})
	}

	item := models.Setting{
		Name:      req.Name,
		ValueType: strings.ToLower(req.ValueType),
		ValueJSON: models.ScalarJSON(datatypes.JSON(req.Value)),
	}
	if err := h.DB.Create(&item).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}
	return c.Status(201).JSON(item)
}

// Patch by id
// @Summary Update setting by id
// @Tags Setting
// @Accept json
// @Produce json
// @Param id path string true "setting id (uuid)"
// @Param body body UpdateSettingRequest true "update setting"
// @Success 200 {object} models.Setting
// @Failure 400 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings/{id} [patch]
func (h *Server) UpdateSettingByID(c fiber.Ctx) error {
	id := c.Params("id")

	var req UpdateSettingRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(APIError{Message: err.Error()})
	}

	var item models.Setting
	if err := h.DB.Where("id = ?", id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(APIError{Message: "setting not found"})
		}
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}

	// 如果传了 value_type，就用新类型；否则沿用旧类型
	// If value_type provided, use it; otherwise keep current type
	valueType := item.ValueType
	if req.ValueType != nil && strings.TrimSpace(*req.ValueType) != "" {
		valueType = strings.ToLower(strings.TrimSpace(*req.ValueType))
	}

	// 若传了 value，则校验并更新
	// If value provided, validate and update
	if len(bytes.TrimSpace(req.Value)) > 0 {
		if err := validateValueTypeAndJSON(valueType, req.Value); err != nil {
			return c.Status(400).JSON(APIError{Message: err.Error()})
		}
		item.ValueType = valueType
		item.ValueJSON = models.ScalarJSON(datatypes.JSON(req.Value))
	} else if req.ValueType != nil {
		// 只更新类型不更新值通常没意义，这里直接报错更安全
		// Updating type without value is usually unsafe -> reject
		return c.Status(400).JSON(APIError{Message: "value is required when updating value_type"})
	}

	if err := h.DB.Save(&item).Error; err != nil {
		return c.Status(500).JSON(APIError{Message: err.Error()})
	}
	return c.JSON(item)
}

// GetSettingByID
// @Summary Get setting by id
// @Description Get a setting by UUID id.
// @Tags Setting
// @Produce json
// @Param id path string true "setting id (uuid)"
// @Success 200 {object} models.Setting
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings/{id} [get]
func (h *Server) GetSettingByID(c fiber.Ctx) error {
	id := c.Params("id")

	var item models.Setting
	if err := h.DB.Where("id = ?", id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(APIError{Message: "setting not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Message: err.Error()})
	}

	return c.JSON(item)
}

// GetSettingByName
// @Summary Get setting by name
// @Description Get a setting by unique name.
// @Tags Setting
// @Produce json
// @Param name path string true "setting name"
// @Success 200 {object} models.Setting
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings/by-name/{name} [get]
func (h *Server) GetSettingByName(c fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Message: "name is required"})
	}

	var item models.Setting
	if err := h.DB.Where("name = ?", name).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(APIError{Message: "setting not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Message: err.Error()})
	}

	return c.JSON(item)
}

// UpsertSettingValueByName
// @Summary Upsert setting by name
// @Description Create if missing; otherwise update value (and optionally value_type).
// @Description value must match value_type: int|string|bool.
// @Tags Setting
// @Accept json
// @Produce json
// @Param name path string true "setting name"
// @Param body body UpsertSettingByNameRequest true "upsert payload"
// @Success 200 {object} models.Setting
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings/by-name/{name} [put]
func (h *Server) UpsertSettingValueByName(c fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Message: "name is required"})
	}

	var req UpsertSettingByNameRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Message: err.Error()})
	}

	// value 一定要有 / value is required
	if len(bytes.TrimSpace(req.Value)) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Message: "value is required"})
	}

	// 事务保证一致性 / Use transaction for consistency
	var out models.Setting
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var item models.Setting
		findErr := tx.Where("name = ?", name).First(&item).Error

		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			// 不存在 -> 创建 / Not found -> create
			if strings.TrimSpace(req.ValueType) == "" {
				// 新建必须提供 value_type / value_type required for create
				return errors.New("value_type is required when creating a new setting")
			}

			vt := strings.ToLower(strings.TrimSpace(req.ValueType))
			if err := validateValueTypeAndJSON(vt, req.Value); err != nil {
				return err
			}

			item = models.Setting{
				Name:      name,
				ValueType: vt,
				ValueJSON: models.ScalarJSON(datatypes.JSON(req.Value)),
			}

			// 这里可能因并发导致 unique 冲突；冲突则再查一次并走更新
			// Unique conflict may happen under concurrency; on conflict, refetch then update.
			if err := tx.Create(&item).Error; err != nil {
				// retry: someone created it first
				var again models.Setting
				if err2 := tx.Where("name = ?", name).First(&again).Error; err2 != nil {
					return err // return original create error if refetch fails
				}
				// update existing
				vt2 := again.ValueType
				if strings.TrimSpace(req.ValueType) != "" {
					vt2 = strings.ToLower(strings.TrimSpace(req.ValueType))
				}
				if err := validateValueTypeAndJSON(vt2, req.Value); err != nil {
					return err
				}
				again.ValueType = vt2
				again.ValueJSON = models.ScalarJSON(datatypes.JSON(req.Value))
				if err := tx.Save(&again).Error; err != nil {
					return err
				}
				out = again
				return nil
			}

			out = item
			return nil
		}

		// 存在 -> 更新 / Found -> update
		vt := item.ValueType
		if strings.TrimSpace(req.ValueType) != "" {
			vt = strings.ToLower(strings.TrimSpace(req.ValueType))
		}
		if err := validateValueTypeAndJSON(vt, req.Value); err != nil {
			return err
		}

		item.ValueType = vt
		item.ValueJSON = models.ScalarJSON(datatypes.JSON(req.Value))

		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		out = item
		return nil
	})

	if err != nil {
		// 这里把校验错误也返回 400 / validation errors -> 400
		if strings.Contains(err.Error(), "value_type") ||
			strings.Contains(err.Error(), "value must") ||
			strings.Contains(err.Error(), "value is required") {
			return c.Status(fiber.StatusBadRequest).JSON(APIError{Message: err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Message: err.Error()})
	}

	return c.JSON(out)
}

// DeleteSettingByID
// @Summary Delete setting by id
// @Description Delete a setting by UUID id.
// @Tags Setting
// @Produce json
// @Param id path string true "setting id (uuid)"
// @Success 204
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /api/v1/settings/{id} [delete]
func (h *Server) DeleteSettingByID(c fiber.Ctx) error {
	id := c.Params("id")

	res := h.DB.Where("id = ?", id).Delete(&models.Setting{})
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Message: res.Error.Error()})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Message: "setting not found"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
