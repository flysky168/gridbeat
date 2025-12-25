package pluginapi

import (
	"fmt"
	"sync"
)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

// RegisterFactory 注册一个插件工厂（内置和 .so 插件都用这个）
// RegisterFactory registers a plugin factory (for both built-in and .so plugins).
func RegisterFactory(f Factory) {
	mu.Lock()
	defer mu.Unlock()

	t := f.Type()
	if t == "" {
		panic("pluginapi: empty factory type")
	}
	if _, exists := factories[t]; exists {
		panic(fmt.Sprintf("pluginapi: duplicate factory type %q", t))
	}
	factories[t] = f
}

// GetFactory 按类型名获取工厂
// GetFactory returns the factory by type name.
func GetFactory(t string) (Factory, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := factories[t]
	return f, ok
}

// AllFactories 返回所有已注册的工厂
// AllFactories returns all registered factories.
func AllFactories() []Factory {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Factory, 0, len(factories))
	for _, f := range factories {
		out = append(out, f)
	}
	return out
}
