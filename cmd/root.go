package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/pluginapi"
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
	rootCmd.PersistentFlags().StringVarP(&core.Gconfig.Plugins, "plugins", "", "./plugins", "load specified plugins folder")

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig() {

	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	viper.SetEnvPrefix(version.ProgramName)

	viper.AutomaticEnv()

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

	// get wd
	cdir := WorkDir()

	if viper.GetString("log-path") == "" {
		viper.Set("log-path", cdir)
	}

	if viper.GetString("data-path") == "" {
		viper.Set("data-path", cdir)
	}

	if viper.GetString("extra-path") == "" {
		viper.Set("extra-path", cdir)
	}

	if viper.GetString("pid") == "" {
		viper.Set("pid", cdir+"/"+version.ProgramName+".pid")
	}

	if err := viper.Unmarshal(&core.Gconfig); err != nil {
		cobra.CheckErr(err)
	}

	if abs, err := filepath.Abs(core.Gconfig.LogPath); err != nil {
		cobra.CheckErr(err)
	} else {
		core.Gconfig.LogPath = abs
	}

	if abs, err := filepath.Abs(core.Gconfig.DataPath); err != nil {
		cobra.CheckErr(err)
	} else {
		core.Gconfig.DataPath = abs
	}

	if abs, err := filepath.Abs(core.Gconfig.ExtraPath); err != nil {
		cobra.CheckErr(err)
	} else {
		core.Gconfig.ExtraPath = abs
	}

	if core.Gconfig.MQTT.Host == "" {
		core.Gconfig.MQTT.Host = "localhost"
		viper.Set("mqtt.host", "localhost")

	}
	if core.Gconfig.MQTT.Port == 0 {
		core.Gconfig.MQTT.Port = 1883
		viper.Set("mqtt.port", "1883")
	}

	if core.Gconfig.HTTP.Port == 0 {
		core.Gconfig.HTTP.Port = 8080
		viper.Set("http.port", "8080")
	}
	if core.Gconfig.HTTPS.Port == 0 {
		core.Gconfig.HTTPS.Port = 8443
		viper.Set("https.port", "8443")
	}

	/*
		if loaded, err := loadAllPlugins(core.Gconfig.Plugins); err != nil {
			log.Fatalf("load plugins error: %v", err)
		} else {
			fmt.Printf("loaded %d plugins:\n", len(loaded))
			for _, p := range loaded {
				fmt.Println(" -", p.Name())
			}
		}
	*/
}

func WorkDir() string {
	dir, _ := os.Getwd()
	return dir
}

func ExeDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func loadAllPlugins(dir string) ([]pluginapi.Plugin, error) {
	var result []pluginapi.Plugin

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".so" {
			continue
		}

		path := filepath.Join(dir, e.Name())
		log.Printf("loading plugin: %s", path)

		p, err := plugin.Open(path)
		if err != nil {
			log.Printf("  open failed: %v", err)
			continue
		}

		// 约定每个插件都导出名为 "Plugin" 的符号
		sym, err := p.Lookup("Plugin")
		if err != nil {
			log.Printf("  lookup Plugin failed: %v", err)
			continue
		}

		pl, ok := sym.(pluginapi.Plugin)
		if !ok {
			log.Printf("  symbol Plugin type mismatch (not pluginapi.Plugin)")
			continue
		}

		// 调用插件初始化
		if err := pl.Init(); err != nil {
			log.Printf("  init failed: %v", err)
			continue
		}

		result = append(result, pl)
	}

	return result, nil
}
