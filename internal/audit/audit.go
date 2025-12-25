package audit

import (
	"encoding/json"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/auth"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

// Write writes an audit log entry.
// Write 写入一条审计日志。
func Write(db *gorm.DB, c fiber.Ctx, user auth.ContextUser, action, resource string, detail any) {
	var detailStr string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailStr = string(b)
		}
	}

	_ = db.Create(&models.AuditLog{
		UserID:    user.ID,
		Username:  user.Username,
		Action:    action,
		Resource:  resource,
		Method:    c.Method(),
		Path:      c.Path(),
		Detail:    detailStr,
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		CreatedAt: time.Now(),
	}).Error
}

// SystemWrite writes audit logs for system actions (no user).
// SystemWrite 写入系统行为审计（无用户）。
func SystemWrite(db *gorm.DB, action, resource, method, path string, detail any) {
	var detailStr string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailStr = string(b)
		}
	}
	_ = db.Create(&models.AuditLog{
		Username:  "system",
		Action:    action,
		Resource:  resource,
		Method:    method,
		Path:      path,
		Detail:    detailStr,
		CreatedAt: time.Now(),
	}).Error
}
