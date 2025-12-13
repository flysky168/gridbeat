package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/model"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type ReopenLogger struct {
	mu sync.Mutex

	accesslogPath string
	runlogPath    string
	mqttlogPath   string
	sqllogPath    string

	accesslogPathFile *os.File
	runlogPathFile    *os.File
	mqttlogPathFile   *os.File
	sqllogPathFile    *os.File

	accessLogger *logrus.Logger
	runLogger    *logrus.Logger
	mqttLogger   *logrus.Logger
	sqlLogger    *logrus.Logger
}

func NewReopenLogger(path string, debug bool) (*ReopenLogger, error) {
	var err error

	if err = os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create dir %s: %w", path, err)
	}

	l := &ReopenLogger{
		accesslogPath: path + "/access.log",
		runlogPath:    path + "/run.log",
		mqttlogPath:   path + "/mqtt.log",
		sqllogPath:    path + "/sql.log",
		accessLogger:  logrus.New(),
		runLogger:     logrus.New(),
		mqttLogger:    logrus.New(),
		sqlLogger:     logrus.New(),
	}

	if l.accesslogPathFile, err = os.OpenFile(l.accesslogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.runlogPathFile, err = os.OpenFile(l.runlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.mqttlogPathFile, err = os.OpenFile(l.mqttlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}
	if l.sqllogPathFile, err = os.OpenFile(l.sqllogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return nil, err
	}

	l.accessLogger.SetOutput(l.accesslogPathFile)
	l.accessLogger.SetFormatter(&AccessLogJSONFormatter{})

	l.runLogger.SetOutput(l.runlogPathFile)
	l.runLogger.SetFormatter(&logrus.JSONFormatter{})
	if debug {
		l.runLogger.SetLevel(logrus.DebugLevel)
		l.runLogger.ReportCaller = true
	}

	l.mqttLogger.SetOutput(l.mqttlogPathFile)
	l.mqttLogger.SetFormatter(&logrus.JSONFormatter{})
	if debug {
		l.mqttLogger.SetLevel(logrus.DebugLevel)
		l.mqttLogger.ReportCaller = true
	}

	l.sqlLogger.SetOutput(l.sqllogPathFile)
	l.sqlLogger.SetFormatter(&logrus.JSONFormatter{})
	if debug {
		l.sqlLogger.SetLevel(logrus.DebugLevel)
		l.sqlLogger.ReportCaller = true
	}

	return l, nil
}

func (l *ReopenLogger) Reopen() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.accesslogPathFile != nil {
		_ = l.accesslogPathFile.Close()
	}

	if l.runlogPathFile != nil {
		_ = l.runlogPathFile.Close()
	}

	if l.mqttlogPathFile != nil {
		_ = l.mqttlogPathFile.Close()
	}

	if l.sqllogPathFile != nil {
		_ = l.sqllogPathFile.Close()
	}

	var err error
	if l.accesslogPathFile, err = os.OpenFile(l.accesslogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.runlogPathFile, err = os.OpenFile(l.runlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.mqttlogPathFile, err = os.OpenFile(l.mqttlogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}
	if l.sqllogPathFile, err = os.OpenFile(l.sqllogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err != nil {
		return err
	}

	l.accessLogger.SetOutput(l.accesslogPathFile)
	l.runLogger.SetOutput(l.runlogPathFile)
	l.mqttLogger.SetOutput(l.mqttlogPathFile)
	l.sqlLogger.SetOutput(l.sqllogPathFile)

	return nil
}

func (l *ReopenLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.accesslogPathFile != nil {
		_ = l.accesslogPathFile.Close()
	}

	if l.runlogPathFile != nil {
		_ = l.runlogPathFile.Close()
	}

	if l.mqttlogPathFile != nil {
		_ = l.mqttlogPathFile.Close()
	}

	if l.sqllogPathFile != nil {
		_ = l.sqllogPathFile.Close()
	}
}

func createPidFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(strconv.Itoa(os.Getpid()))
	f.Close()
	return err
}

func removePidFile(path string) {
	_ = os.Remove(path)
}

func init() {
	rootCmd.AddCommand(serverCmd)

	flags := serverCmd.Flags()
	flags.BoolVar(&core.Gconfig.DisableAuth, "disable_auth", false, "disable http api auth")
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

func handleSignals(logger *ReopenLogger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGINT)

	for sig := range ch {
		switch sig {
		case syscall.SIGUSR1:
			log.Println("received SIGUSR1, reopening log file")
			if err := logger.Reopen(); err != nil {
				log.Printf("reopen log failed: %v\n", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			log.Println("exiting")
			logger.Close()
			os.Exit(0)
		}
	}
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {

		var err error
		var logger *ReopenLogger

		if logger, err = NewReopenLogger(core.Gconfig.LogPath, core.Gconfig.Debug); err != nil {
			cobra.CheckErr(err)
		}
		defer logger.Close()

		fmt.Printf("use log path: %s\n", core.Gconfig.LogPath)

		if err = model.InitDB(core.Gconfig.DataPath, "app.db", logger.sqlLogger); err != nil {
			logger.runLogger.Fatal(err)
		}

		if err := createPidFile(core.Gconfig.PID); err != nil {
			cobra.CheckErr(fmt.Errorf("already running? %w", err))
		}
		defer removePidFile(core.Gconfig.PID)

		go handleSignals(logger)

		// mqtt
		var server *mqtt.Server
		if server, err = core.ServerMQTT(logger.mqttLogger); err != nil {
			logger.mqttLogger.Fatal(err)
			return err
		}
		if err := server.Serve(); err != nil {
			logger.mqttLogger.Fatal(err)
			return err
		}

		go core.ServerHTTP(server, logger.runLogger, logger.accessLogger)

		go core.ServerPlugins(server, logger.runLogger)

		for {
			time.Sleep(1000 * time.Second)
		}

		return nil
	},
}
