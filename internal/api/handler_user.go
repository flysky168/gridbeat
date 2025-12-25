package api

import (
	"errors"
	"strconv"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/audit"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/internal/response"
	"github.com/fluxionwatt/gridbeat/internal/util"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// AdminCreateUserRequest creates a user.
// AdminCreateUserRequest 创建用户请求体。
type AdminCreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminUserItem is user list item.
// AdminUserItem 是用户列表项。
type AdminUserItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	IsRoot    bool      `json:"is_root"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminListUsers lists users (root only).
// AdminListUsers 列出用户（仅 root）。
//
// @Summary List users / 列出用户
// @Description Root user lists all users.
// @Description root 用户查看所有用户。
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Envelope[[]AdminUserItem]
// @Router /api/v1/admin/users [get]
func (s *Server) AdminListUsers(c fiber.Ctx) error {
	var users []models.User
	if err := s.DB.Order("id asc").Find(&users).Error; err != nil {
		return response.Internal(c, "db error")
	}
	out := make([]AdminUserItem, 0, len(users))
	for _, u := range users {
		out = append(out, AdminUserItem{
			ID:        u.ID,
			Username:  u.Username,
			IsRoot:    u.IsRoot,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}
	return response.OK(c, out)
}

// AdminCreateUser creates a new user (root only).
// AdminCreateUser 创建新用户（仅 root）。
//
// @Summary Create user / 创建用户
// @Description Root creates a normal user.
// @Description root 创建普通用户。
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AdminCreateUserRequest true "request / 请求"
// @Success 200 {object} response.Envelope[AdminUserItem]
// @Failure 409 {object} response.Envelope[any]
// @Router /api/v1/admin/users [post]
func (s *Server) AdminCreateUser(c fiber.Ctx) error {
	admin := MustUser(c)

	var req AdminCreateUserRequest
	if err := c.Bind().JSON(req); err != nil {
		return response.BadRequest(c, "invalid json")
	}
	if req.Username == "" || req.Password == "" {
		return response.BadRequest(c, "username/password required")
	}
	if req.Username == "root" {
		return response.Conflict(c, "root already exists")
	}

	var exists models.User
	if err := s.DB.Where("username = ?", req.Username).First(&exists).Error; err == nil {
		return response.Conflict(c, "username exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.Internal(c, "db error")
	}

	hash, err := util.HashPassword(req.Password)
	if err != nil {
		return response.BadRequest(c, "password invalid")
	}

	now := time.Now()
	u := models.User{
		Username:     req.Username,
		PasswordHash: hash,
		IsRoot:       false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.DB.Create(&u).Error; err != nil {
		return response.Internal(c, "create user failed")
	}

	audit.Write(s.DB, c, admin, "create_user", "user", fiber.Map{"user_id": u.ID, "username": u.Username})
	return response.OK(c, AdminUserItem{ID: u.ID, Username: u.Username, IsRoot: u.IsRoot, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt})
}

// AdminResetPasswordRequest is request body for resetting password.
// AdminResetPasswordRequest 是重置密码请求体。
type AdminResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// AdminResetUserPassword resets any user's password (root only).
// AdminResetUserPassword root 重置任意用户密码。
//
// @Summary Reset user password / 重置用户密码
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "user id / 用户ID"
// @Param body body AdminResetPasswordRequest true "request / 请求"
// @Success 200 {object} response.Envelope[any]
// @Router /api/v1/admin/users/{id}/password [put]
func (s *Server) AdminResetUserPassword(c fiber.Ctx) error {
	admin := MustUser(c)

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	var req AdminResetPasswordRequest
	if err := c.Bind().JSON(req); err != nil {
		return response.BadRequest(c, "invalid json")
	}
	if req.NewPassword == "" {
		return response.BadRequest(c, "new_password required")
	}

	var u models.User
	if err := s.DB.First(&u, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		return response.Internal(c, "db error")
	}

	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return response.BadRequest(c, "password invalid")
	}

	if err := s.DB.Model(&u).Update("password_hash", hash).Error; err != nil {
		return response.Internal(c, "update failed")
	}

	// Revoke all tokens after admin password reset.
	// 管理员重置密码后撤销该用户所有 token。
	now := time.Now()
	_ = s.DB.Model(&models.AuthToken{}).
		Where("user_id = ? AND revoked_at IS NULL", u.ID).
		Updates(map[string]any{"revoked_at": &now, "revoked_by": admin.ID}).Error

	audit.Write(s.DB, c, admin, "reset_password", "user", fiber.Map{"user_id": u.ID, "username": u.Username})
	return response.OK(c, fiber.Map{"reset": true})
}

// AdminDeleteUser deletes a user (root only), but cannot delete root.
// AdminDeleteUser 删除用户（仅 root），但不能删除 root。
//
// @Summary Delete user / 删除用户
// @Description Root deletes a user, but root itself is protected.
// @Description root 删除用户，但 root 自身受保护不可删除。
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "user id / 用户ID"
// @Success 200 {object} response.Envelope[any]
// @Failure 403 {object} response.Envelope[any]
// @Router /api/v1/admin/users/{id} [delete]
func (s *Server) AdminDeleteUser(c fiber.Ctx) error {
	admin := MustUser(c)

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.BadRequest(c, "invalid id")
	}

	var u models.User
	if err := s.DB.First(&u, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NotFound(c, "user not found")
		}
		return response.Internal(c, "db error")
	}

	// Protect root from deletion.
	// 保护 root 不被删除。
	if u.Username == "root" || u.IsRoot {
		return response.Forbidden(c, "cannot delete root")
	}

	if err := s.DB.Delete(&u).Error; err != nil {
		return response.Internal(c, "delete failed")
	}

	audit.Write(s.DB, c, admin, "delete_user", "user", fiber.Map{"user_id": u.ID, "username": u.Username})
	return response.OK(c, fiber.Map{"deleted": true})
}
