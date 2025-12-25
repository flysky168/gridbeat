package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/internal/auth"
	"github.com/fluxionwatt/gridbeat/internal/db"
	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/pluginapi"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	http "github.com/fluxionwatt/gridbeat/core/http"
	"github.com/fluxionwatt/gridbeat/core/plugin/cmbus"
	_ "github.com/fluxionwatt/gridbeat/core/plugin/goose"
	"github.com/fluxionwatt/gridbeat/core/plugin/mbus"
	_ "github.com/fluxionwatt/gridbeat/core/plugin/stream"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	flags := serverCmd.Flags()
	flags.BoolVar(&core.Gconfig.DisableAuth, "disable_auth", false, "disable http api auth")
	flags.BoolVar(&core.Gconfig.Simulator, "simulator", false, "enable simulator client")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run server",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		cfg := &core.Gconfig

		// Enable/disable auth globally.
		// 全局启用/禁用鉴权。
		auth.NoAuth = core.Gconfig.DisableAuth

		var err error
		var logger *pluginapi.ReopenLogger

		if logger, err = pluginapi.NewReopenLogger(core.Gconfig.LogPath, core.Gconfig.Debug); err != nil {
			cobra.CheckErr(err)
			return
		}

		fmt.Printf("use log path: %s\n", core.Gconfig.LogPath)

		// 从目录加载 .so 插件工厂
		// Optional: load .so plugin factories from a directory.
		// 内置插件已经通过 init() 完成工厂注册
		// Built-in factories are already registered via init() above.
		loadSoFactories(core.Gconfig.Plugins)

		for _, f := range pluginapi.AllFactories() {
			logger.RunLogger.Info(fmt.Sprintf("factories registered %s", f.Type()))
		}

		gdb, err := db.Open(cfg, logger.SqlLogger)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("db open error %w", err))
		}

		if err := models.Migrate(gdb); err != nil {
			cobra.CheckErr(fmt.Errorf("db migrate %w", err))
			return
		}

		// Build registry
		reg := models.NewRegistry()

		// 注册模型（可选：给某个模型设置默认 Preload）
		if err := reg.Register(gdb, models.Channel{} /*, modelutil.WithDefaultPreloads("Profile")*/); err != nil {
			cobra.CheckErr(fmt.Errorf("register channel %w", err))
			return
		}

		/*
			ctx := context.Background()

			// ✅ 只传 table + pkValue
			obj, err := reg.FindByTablePK(ctx, db, "users", 1)
			if err != nil {
				log.Fatal(err)
			}
			u := obj.(*example.User)
			fmt.Println("User:", u.ID, u.Name, u.Email)

			// ✅ 仍然走 Model(&T{})，可选额外 Preload / Scope（不影响你“只传 table+pkValue”）
			obj2, err := reg.FindByTablePKWith(ctx, db, "devices", int64(1),
				modelutil.WithScopes(func(tx *gorm.DB) *gorm.DB {
					return tx.Where("tag = ?", "meter")
				}),
			)
			if err != nil {
				log.Fatal(err)
			}
			d := obj2.(*example.Device)
			fmt.Println("Device:", d.ID, d.SN, d.Tag)

			// ✅ 如果你想拿到非指针的 struct 值
			val, err := reg.FindByTablePKValue(ctx, db, "users", 1)
			if err != nil {
				log.Fatal(err)
			}
			u2 := val.(example.User)
			fmt.Println("User value:", u2.ID, u2.Name)
		*/

		// Ensure root exists ("admin" password by default if created).
		// 确保 root 存在（首次创建默认密码 admin）。
		if err := models.EnsureRootUser(gdb); err != nil {
			cobra.CheckErr(fmt.Errorf("ensure root user %w", err))
			return
		}

		if err := db.SyncSerials(gdb, cfg.Serial); err != nil {
			cobra.CheckErr(fmt.Errorf("sync serials failed %w", err))
			return
		}

		// 初始化默认 setting 数据（只补缺，不覆盖）
		// Seed default settings (insert missing only, do NOT overwrite)
		if err := db.SeedDefaultSettings(gdb); err != nil {
			cobra.CheckErr(fmt.Errorf("seed default settings failed %w", err))
			return
		}

		// mqtt
		var server *mqtt.Server
		if server, err = core.ServerMQTT(logger.MqttLogger); err != nil {
			logger.MqttLogger.Error(err)
			cobra.CheckErr(fmt.Errorf("server  d mqtt %w", err))
			return
		}

		rootCtx, rootCancel := context.WithCancel(context.Background())
		//defer rootCancel()

		var wg sync.WaitGroup

		// 捕获信号 / capture OS signals.
		go func() {
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
					rootCancel()
					log.Println("root cannel")
					wg.Wait()
					log.Println("wait")
					core.RemovePidFile(core.Gconfig.PID)
					logger.Close()
					os.Exit(0)
				}
			}
		}()

		// 宿主环境 / host environment
		env := &pluginapi.HostEnv{
			Logger:    logger,
			DB:        gdb,
			MQTT:      server,
			Conf:      &core.Gconfig,
			PluginLog: logrus.NewEntry(logger.RunLogger),
			WG:        &wg,
		}

		// 带 rootCtx + env 的 InstanceManager
		mgr := core.NewInstanceManager(rootCtx, env)
		defer mgr.DestroyAll() // 进程退出时清理所有实例 / cleanup on shutdown

		cycle := &core.Cycle{
			Logger:       logrus.NewEntry(logger.RunLogger),
			DB:           gdb,
			MQTT:         server,
			Conf:         cfg,
			Mgr:          mgr,
			WG:           &wg,
			AccessLogger: logger.AccessLogger,
		}

		if err := core.CreatePidFile(core.Gconfig.PID); err != nil {
			cobra.CheckErr(fmt.Errorf("already running? %w", err))
		}

		if err := cycle.MQTT.Serve(); err != nil {
			logger.MqttLogger.Error(err)
			cobra.CheckErr(fmt.Errorf("server mqtt start %w", err))
			return
		}

		handler := http.New(http.Config{
			HTTPSDisable: core.Gconfig.HTTPS.Disable,
			HTTPAddress:  ":" + viper.GetString("http.port"),
			HTTPSAddress: ":" + viper.GetString("https.port"),
		})

		handler.Init(rootCtx, cycle)

		var items []models.Channel
		if err := gdb.Order("uuid asc").Find(&items).Error; err != nil {
			cobra.CheckErr(fmt.Errorf("get all channel %w", err))
			return
		}
		for _, channel := range items {

			if core.Gconfig.Simulator {
				if _, err = mgr.Create("cmbus", channel.UUID, cmbus.InstanceConfig{
					Model: channel,
				}); err != nil {
					cobra.CheckErr(fmt.Errorf("mgr create instance %w", err))
				}
			}

			if _, err = mgr.Create("mbus", channel.UUID, mbus.InstanceConfig{
				Model:     channel,
				UnitID:    1,
				Quantity:  1,
				StartAddr: 100,
			}); err != nil {
				cobra.CheckErr(fmt.Errorf("mgr create instance %w", err))
			}
		}

		select {}
	},
}
