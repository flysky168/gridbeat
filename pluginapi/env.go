package pluginapi

import (
	"sync"

	"github.com/fluxionwatt/gridbeat/internal/config"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// HostEnv 表示宿主环境中可注入到插件实例的全局对象
// HostEnv represents global objects from the host that can be injected into plugin instances.
type HostEnv struct {

	// Values：扩展用，可以放任意其他全局对象（DB、metrics、缓存客户端等）
	// Values: extension map for arbitrary global objects (DB, metrics, cache clients, etc).
	//Values map[string]any

	Conf   *config.Config
	DB     *gorm.DB
	Logger *ReopenLogger

	PluginLog logrus.FieldLogger
	MQTT      *mqtt.Server
	WG        *sync.WaitGroup
}

const depsKey = "__global_deps__"

func Inject(db *gorm.DB, deps *HostEnv) *gorm.DB {
	if db == nil || deps == nil {
		return db
	}
	// Set 会返回一个新的 *gorm.DB（session），你可以把它当作全局 gdb 使用
	return db.Set(depsKey, deps)
}

func FromEnv(tx *gorm.DB) *HostEnv {
	if tx == nil {
		return nil
	}
	if v, ok := tx.Get(depsKey); ok {
		if d, ok := v.(*HostEnv); ok {
			return d
		}
	}
	return nil
}
