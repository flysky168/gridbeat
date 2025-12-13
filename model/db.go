package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Gdb *gorm.DB

type LogrusLogger struct {
	Log           *logrus.Entry
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

func NewLogrusLogger(log *logrus.Logger) *LogrusLogger {
	return &LogrusLogger{
		Log:           log.WithField("module", "gorm"),
		LogLevel:      logger.Info,            // 自己控制级别
		SlowThreshold: 200 * time.Millisecond, // 慢查询阈值
	}
}

func (l *LogrusLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *LogrusLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.Log.WithContext(ctx).Infof(msg, args...)
	}
}

func (l *LogrusLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.Log.WithContext(ctx).Warnf(msg, args...)
	}
}

func (l *LogrusLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.Log.WithContext(ctx).Errorf(msg, args...)
	}
}

func (l *LogrusLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	entry := l.Log.WithContext(ctx).WithFields(logrus.Fields{
		"elapsed": elapsed,
		"rows":    rows,
		"sql":     sql,
	})

	switch {
	case err != nil && l.LogLevel >= logger.Error:
		entry.WithError(err).Error("gorm sql error")
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		entry.Warn("gorm slow sql")
	case l.LogLevel >= logger.Info:
		entry.Info("gorm sql")
	}
}

func InitDB(path string, filename string, errorLogger *logrus.Logger) error {
	gormLogger := NewLogrusLogger(errorLogger)
	gormLogger.LogLevel = logger.LogLevel(logrus.DebugLevel)
	gormLogger.SlowThreshold = 500 * time.Millisecond

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create dir %s: %w", path, err)
	}

	db, err := gorm.Open(sqlite.Open(path+"/"+filename), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Log{})

	db.AutoMigrate(&Channel{}, &Device{}, &DevicePoint{}, &Array{}, &Site{})

	ctx := context.Background()
	if _, err := gorm.G[User](db).Take(ctx); errors.Is(err, gorm.ErrRecordNotFound) {
		var d = User{}
		gorm.G[User](db).Create(ctx, d.GetInitData())
	}

	Gdb = db

	return nil
}
