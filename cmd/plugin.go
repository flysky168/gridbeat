package cmd

import (
	"log"
	"os"
	"path/filepath"
	"plugin"

	"github.com/fluxionwatt/gridbeat/pluginapi"
	"gopkg.in/yaml.v3"
)

// PluginInstanceConfig 表示 YAML 中的一个实例配置
// PluginInstanceConfig describes one plugin instance in YAML.
type PluginInstanceConfig struct {
	ID     string         `yaml:"id"`      // 实例 ID / instance ID
	Config map[string]any `yaml:",inline"` // 其他字段合并进 map / all other fields go into this map
}

// AppConfig 顶层配置结构
// AppConfig is the top-level configuration structure.
type AppConfig struct {
	// 形如：
	// plugins:
	//   goose:
	//     - id: goose-1
	//       dsn: "postgres://..."
	//       flush_interval: "2s"
	//
	// Plugins maps plugin type -> list of instance configs.
	Plugins map[string][]PluginInstanceConfig `yaml:"plugins"`
}

// loadConfig 从指定路径加载 YAML 配置
// loadConfig loads YAML configuration from the given path.
func loadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Plugins == nil {
		cfg.Plugins = make(map[string][]PluginInstanceConfig)
	}
	return &cfg, nil
}

// loadSoFactories 扫描目录中的 .so 文件，加载并注册工厂
// loadSoFactories scans the given directory for .so files, loads and registers factories.
//
// 约定 .so 内部导出符号：
//
//	var Factory pluginapi.Factory = &SomeFactory{}
//
// plugin.Lookup("Factory") 会返回 *pluginapi.Factory（指向变量的指针）
// plugin.Lookup("Factory") returns *pluginapi.Factory (pointer to the variable).
func loadSoFactories(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("read so dir %s: %v", dir, err)
		return
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".so" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		log.Printf("loading so plugin: %s", path)

		p, err := plugin.Open(path)
		if err != nil {
			log.Printf("  open failed: %v", err)
			continue
		}

		sym, err := p.Lookup("Factory")
		if err != nil {
			log.Printf("  lookup Factory failed: %v", err)
			continue
		}

		//log.Printf("  sym type: %T", sym)

		// 变量导出 => *pluginapi.Factory
		// Variable export => *pluginapi.Factory.
		ptr, ok := sym.(*pluginapi.Factory)
		if !ok {
			log.Printf("  symbol Factory type mismatch (not *pluginapi.Factory)")
			continue
		}

		f := *ptr
		log.Printf("  register factory type=%s from %s", f.Type(), path)
		pluginapi.RegisterFactory(f)
	}
}
