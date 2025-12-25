package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/util"
	"gorm.io/gorm"
)

// Migrate runs auto migrations.
// Migrate 执行数据库自动迁移。
func Migrate(db *gorm.DB) error {

	if err := db.AutoMigrate(&Channel{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&DeviceType{}, &DeviceTypePoint{}, &Device{}); err != nil {
		return err
	}

	if err := db.AutoMigrate(&User{}, &AuthToken{}, &AuditLog{}, &Setting{}); err != nil {
		return err
	}

	return nil
}

// EnsureRootUser ensures the super user "root" exists.
// EnsureRootUser 确保超级用户 root 存在。
func EnsureRootUser(db *gorm.DB) error {
	var u User
	err := db.Where("username = ?", "root").First(&u).Error
	if err == nil {
		// Make sure it's protected.
		// 确保 root 标记为超级用户。
		if !u.IsRoot {
			u.IsRoot = true
			return db.Save(&u).Error
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("query root user failed: %w", err)
	}

	hash, err := util.HashPassword("admin")
	if err != nil {
		return err
	}

	root := User{
		Username:     "root",
		PasswordHash: hash,
		IsRoot:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return db.Create(&root).Error
}
