package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/config"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Open opens sqlite database under cfg.App.DB.Dir.
// Open 在 cfg.App.DB.Dir 下打开 sqlite 数据库。
func Open(cfg *config.Config, errorLogger *logrus.Logger) (*gorm.DB, error) {
	if err := config.EnsureDBDir(cfg.DataPath); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.DataPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create dir %s: %w", cfg.DataPath, err)
	}

	// Use a conservative logger level in production.
	// 生产环境建议降低日志等级。
	gormCfg := &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Warn),
		Logger: models.NewLogrusLogger(errorLogger),
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(cfg.DataPath, "app.db")), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open sqlite failed: %w", err)
	}

	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %v", err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func SeedDefaultSettings(gdb *gorm.DB) error {
	// helper: must marshal JSON (panic-free pattern)
	// 工具函数：把任意值转成 JSON bytes
	mustJSON := func(v any) models.ScalarJSON {
		b, _ := json.Marshal(v)
		return models.ScalarJSON(b)
	}

	defaults := []models.Setting{
		{Name: "site_name", ValueType: "string", ValueJSON: mustJSON("Energy Gateway")},
		{Name: "timezone", ValueType: "string", ValueJSON: mustJSON("Asia/Tokyo")},
		{Name: "lang", ValueType: "string", ValueJSON: mustJSON("zh-CN")},
		{Name: "log_level", ValueType: "string", ValueJSON: mustJSON("info")},
		{Name: "feature_swagger", ValueType: "bool", ValueJSON: mustJSON(true)},
		{Name: "sampling_interval_sec", ValueType: "int", ValueJSON: mustJSON(10)},
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, d := range defaults {
			var existing models.Setting
			err := tx.Where("name = ?", d.Name).First(&existing).Error
			if err == nil {
				continue
			}
			if err != gorm.ErrRecordNotFound {
				fmt.Println("fdasfdsfas", d.Name)
				return err
			}
			if err := tx.Create(&d).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SyncSerials syncs the serial table with startup device list / 根据启动参数同步 serial 表
//
// Rules / 规则：
// 1) Devices present in list:
//   - insert if missing / 不存在则新增
//   - update UpdatedAt if exists / 存在则更新时间（也可更新其他字段）
//
// 2) Devices NOT present in list: delete the record / 不在列表中的设备删除记录
func SyncSerials(gdb *gorm.DB, devices []config.Serial) error {
	// Normalize, trim, deduplicate / 规范化、去空格、去重
	set := make(map[string]struct{}, len(devices))
	normalized := make([]string, 0, len(devices))
	normalized2 := make([]string, 0, len(devices))
	for _, s := range devices {
		d := s.Device

		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := set[d]; ok {
			continue
		}
		set[d] = struct{}{}
		normalized = append(normalized, d)
		normalized2 = append(normalized2, s.Device2)
	}

	now := time.Now()

	return gdb.Transaction(func(tx *gorm.DB) error {
		// Load existing records / 读取现有记录
		var existing []models.Channel
		if err := tx.Find(&existing).Error; err != nil {
			return fmt.Errorf("query serials failed: %w", err)
		}
		existingSet := make(map[string]models.Channel, len(existing))
		for _, s := range existing {
			existingSet[s.Device] = s
		}

		// Upsert for present devices / 对存在的设备进行 upsert
		for i, dev := range normalized {
			if s, ok := existingSet[dev]; ok {
				// Update updated_at to reflect current boot config / 更新时间反映当前启动配置
				if err := tx.Model(&models.Channel{}).
					Where("id = ?", s.ID).
					Updates(map[string]any{"updated_at": now}).Error; err != nil {
					return fmt.Errorf("update serial failed: %w", err)
				}
			} else {
				// Insert new row / 新增记录
				if err := tx.Create(models.GetDefaultSerialRow(dev, normalized2[i])).Error; err != nil {
					return fmt.Errorf("create serial failed: %w", err)
				}
			}
		}

		// Delete records not in list / 删除不在列表中的记录
		// If startup list is empty, it means delete all records / 如果启动参数为空，则删除全部记录
		if len(normalized) == 0 {
			if err := tx.Where("1 = 1").Delete(&models.Channel{}).Error; err != nil {
				return fmt.Errorf("delete all serials failed: %w", err)
			}
			return nil
		}

		if err := tx.Where("device NOT IN ?", normalized).Delete(&models.Channel{}).Error; err != nil {
			return fmt.Errorf("delete missing serials failed: %w", err)
		}
		return nil
	})
}
