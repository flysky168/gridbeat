package models

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

type LogrusLogger struct {
	Log           *logrus.Entry
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

func NewLogrusLogger(log *logrus.Logger) *LogrusLogger {
	return &LogrusLogger{
		Log:           log.WithField("module", "gorm"),
		LogLevel:      logger.LogLevel(log.Level),
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
