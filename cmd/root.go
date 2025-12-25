package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   version.ProgramName,
		Short: "An open-source software for data acquisition and monitoring",
		Long: `GridBeat is an open-source software for data acquisition and monitoring 
		of solar photovoltaic systems, energy storage systems, and charging piles.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $PWD/"+version.ProgramName+".yaml)")
	rootCmd.PersistentFlags().BoolVar(&core.Gconfig.Debug, "debug", false, "debug mode")
	rootCmd.PersistentFlags().StringVarP(&core.Gconfig.Plugins, "plugins", "", "./plugins", "directory from which loads plugin lib files")

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig() {

	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	viper.BindPFlag("plugins", rootCmd.Flags().Lookup("plugins"))

	// Environment variables: GRIDBEAT_AUTH_JWT_SECRET ...
	viper.SetEnvPrefix(version.ProgramName)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// get wd
	cdir := WorkDir()

	// Defaults / 默认值
	viper.SetDefault("plugins", "./plugins")
	viper.SetDefault("data-path", cdir)
	viper.SetDefault("log-path", cdir)
	viper.SetDefault("extra-path", cdir)
	viper.SetDefault("pid", cdir+"/"+version.ProgramName+".pid")

	viper.SetDefault("auth.jwt.secret", "change-me")
	viper.SetDefault("auth.jwt.issuer", version.ProgramName)
	viper.SetDefault("auth.web.idle_minutes", 30)
	viper.SetDefault("audit.retention_days", 120)
	viper.SetDefault("mqtt.host", "localhost")
	viper.SetDefault("mqtt.port", "1883")
	viper.SetDefault("http.port", "8080")
	viper.SetDefault("https.port", "8443")

	if cfgFile != "" {
		viper.SetConfigType("yaml")
		viper.SetConfigFile(cfgFile)

		viper.OnConfigChange(func(e fsnotify.Event) {
			//errorLogger.Info("Config file changed:" + e.Name)
		})

		viper.WatchConfig()

		if err := viper.ReadInConfig(); err == nil {
			//fmt.Println("Using config file:", viper.ConfigFileUsed())
		} else {
			// return fmt.Errorf("fatal error config file: %w", err)
			cobra.CheckErr(fmt.Errorf("fatal error config file: %w", err))
		}
	} else {
		// Search config in home directory with name ".cobra" (without extension).
		//viper.SetConfigName(version.ProgramName)
		//viper.AddConfigPath("./config")
		//viper.AddConfigPath("./")
		//viper.AddConfigPath("/etc/" + version.ProgramName + "/")
	}

	if err := viper.Unmarshal(&core.Gconfig); err != nil {
		cobra.CheckErr(err)
	}

	cfg := &core.Gconfig

	// Normalize / 规范化
	cfg.LogPath = filepath.Clean(cfg.LogPath)
	if cfg.Auth.Web.IdleMinutes <= 0 {
		cfg.Auth.Web.IdleMinutes = 30
	}
	if cfg.Audit.RetentionDays <= 0 {
		cfg.Audit.RetentionDays = 120
	}
	if strings.TrimSpace(cfg.Auth.JWT.Issuer) == "" {
		cfg.Auth.JWT.Issuer = version.ProgramName
	}

	if abs, err := filepath.Abs(cfg.LogPath); err != nil {
		cobra.CheckErr(err)
	} else {
		cfg.LogPath = abs
	}

	if abs, err := filepath.Abs(cfg.DataPath); err != nil {
		cobra.CheckErr(err)
	} else {
		cfg.DataPath = abs
	}

	if abs, err := filepath.Abs(cfg.ExtraPath); err != nil {
		cobra.CheckErr(err)
	} else {
		cfg.ExtraPath = abs
	}

	if abs, err := filepath.Abs(cfg.Plugins); err != nil {
		cobra.CheckErr(err)
	} else {
		cfg.Plugins = abs
	}
}

func WorkDir() string {
	dir, _ := os.Getwd()
	return dir
}

func ExeDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}
