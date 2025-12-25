package pluginapi

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type ReopenLogger struct {
	MU sync.Mutex

	AccesslogPath string
	RunlogPath    string
	MqttlogPath   string
	SqllogPath    string

	AccesslogPathFile *os.File
	RunlogPathFile    *os.File
	MqttlogPathFile   *os.File
	SqllogPathFile    *os.File

	AccessLogger *logrus.Logger
	RunLogger    *logrus.Logger
	MqttLogger   *logrus.Logger
	SqlLogger    *logrus.Logger
}

func NewReopenLogger(path string, debug bool) (*ReopenLogger, error) {
	var err error

	if err = os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create dir %s: %w", path, err)
	}

	l := &ReopenLogger{
		AccesslogPath: path + "/access.log",
		RunlogPath:    path + "/run.log",
		MqttlogPath:   path + "/mqtt.log",
		SqllogPath:    path + "/sql.log",
		AccessLogger:  logrus.New(),
		RunLogger:     logrus.New(),
		MqttLogger:    logrus.New(),
		SqlLogger:     logrus.New(),
	}

	if l.AccesslogPathFile, err = os.OpenFile(l.AccesslogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.RunlogPathFile, err = os.OpenFile(l.RunlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.MqttlogPathFile, err = os.OpenFile(l.MqttlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.SqllogPathFile, err = os.OpenFile(l.SqllogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}

	l.AccessLogger.SetOutput(l.AccesslogPathFile)
	l.AccessLogger.SetFormatter(&AccessLogJSONFormatter{})

	l.RunLogger.SetOutput(l.RunlogPathFile)
	l.RunLogger.SetFormatter(&logrus.JSONFormatter{})
	if debug {
		l.RunLogger.SetLevel(logrus.DebugLevel)
		l.RunLogger.ReportCaller = true
	}

	l.MqttLogger.SetOutput(l.MqttlogPathFile)
	l.MqttLogger.SetFormatter(&logrus.JSONFormatter{})
	if debug {
		l.MqttLogger.SetLevel(logrus.DebugLevel)
		l.MqttLogger.ReportCaller = true
	}

	l.SqlLogger.SetOutput(l.SqllogPathFile)
	l.SqlLogger.SetFormatter(&logrus.TextFormatter{})
	if debug {
		l.SqlLogger.SetLevel(logrus.DebugLevel)
		l.SqlLogger.ReportCaller = true
	}

	return l, nil
}

func (l *ReopenLogger) Reopen() error {
	l.MU.Lock()
	defer l.MU.Unlock()

	if l.AccesslogPathFile != nil {
		_ = l.AccesslogPathFile.Close()
	}

	if l.RunlogPathFile != nil {
		_ = l.RunlogPathFile.Close()
	}

	if l.MqttlogPathFile != nil {
		_ = l.MqttlogPathFile.Close()
	}

	if l.SqllogPathFile != nil {
		_ = l.SqllogPathFile.Close()
	}

	var err error
	if l.AccesslogPathFile, err = os.OpenFile(l.AccesslogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.RunlogPathFile, err = os.OpenFile(l.RunlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.MqttlogPathFile, err = os.OpenFile(l.MqttlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.SqllogPathFile, err = os.OpenFile(l.SqllogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}

	l.AccessLogger.SetOutput(l.AccesslogPathFile)
	l.RunLogger.SetOutput(l.RunlogPathFile)
	l.MqttLogger.SetOutput(l.MqttlogPathFile)
	l.SqlLogger.SetOutput(l.SqllogPathFile)

	return nil
}

func (l *ReopenLogger) Close() {
	l.MU.Lock()
	defer l.MU.Unlock()

	if l.AccesslogPathFile != nil {
		_ = l.AccesslogPathFile.Close()
	}

	if l.RunlogPathFile != nil {
		_ = l.RunlogPathFile.Close()
	}

	if l.MqttlogPathFile != nil {
		_ = l.MqttlogPathFile.Close()
	}

	if l.SqllogPathFile != nil {
		_ = l.SqllogPathFile.Close()
	}
}

type AccessLogJSONFormatter struct{}

func (f *AccessLogJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(map[string]interface{}, len(entry.Data))

	for k, v := range entry.Data {
		data[k] = v
	}

	data["time"] = entry.Time.Format("02/Jan/2006:15:04:05 -0700")

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
