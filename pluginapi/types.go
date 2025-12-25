package pluginapi

import "context"

// InstanceConfig 保存单个实例的配置（通用 KV）
// InstanceConfig holds configuration for a single plugin instance (generic key-value).
// type InstanceConfig map[string]any
type InstanceConfig any

// Instance 表示某个插件类型的一条运行实例
// Instance represents a single runtime instance of a plugin type.
type Instance interface {
	// 返回唯一实例 ID，例如 "goose-1"
	// Returns unique instance ID, e.g. "goose-1".
	ID() string

	// 返回插件类型名，例如 "goose"
	// Returns plugin type, e.g. "goose".
	Type() string

	// Init 使用给定的父 ctx 和 HostEnv 做初始化
	// Init uses the given parent ctx and HostEnv for initialization.
	// - parent: 实例级父 context，一般来自系统根 ctx 或更上层的租户 ctx
	//           parent: per-instance parent context, usually derived from system root or tenant ctx.
	// - env: 宿主传下来的全局环境（logger、metrics 等）
	//        env: host environment (logger, metrics, etc.) passed from the host.
	Init(parent context.Context, env *HostEnv) error

	// 销毁前由宿主调用，负责释放资源（停 goroutine / 关连接等）
	// Called before destruction to free all resources.
	Close() error

	// UpdateConfig 用于应用新的配置，实现热更新
	// UpdateConfig applies a new configuration (for hot-reload).
	UpdateConfig(cfg InstanceConfig) error

	Get() any
}

// Factory 表示某个插件类型（驱动），负责创建多个实例
// Factory represents a plugin type (driver), responsible for creating instances.
type Factory interface {
	// 返回插件类型名（全局唯一），例如 "goose"
	// Returns plugin type name (globally unique), e.g. "goose".
	Type() string

	// 根据实例 ID + 配置创建实例（不带 ctx/env，纯构造）
	// Creates a new instance with given ID and config (pure construction, no ctx/env yet).
	New(id string, cfg InstanceConfig) (Instance, error)
}
