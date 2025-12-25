package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxionwatt/gridbeat/pluginapi"
)

// InstanceManager 管理进程中所有插件实例
// InstanceManager manages all plugin instances in the process.
type InstanceManager struct {
	mu        sync.RWMutex
	instances map[string]map[string]pluginapi.Instance // type -> id -> instance
	rootCtx   context.Context                          // 系统根 ctx / system root context
	env       *pluginapi.HostEnv                       // 宿主环境（logger 等）/ host environment (logger, etc.)
}

// NewInstanceManager 创建新的实例管理器，如果 rootCtx 为 nil 则使用 context.Background()
// NewInstanceManager creates a new instance manager; if rootCtx is nil, uses context.Background().
func NewInstanceManager(rootCtx context.Context, env *pluginapi.HostEnv) *InstanceManager {
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return &InstanceManager{
		instances: make(map[string]map[string]pluginapi.Instance),
		rootCtx:   rootCtx,
		env:       env,
	}
}

// CreateWithContext 使用指定 parentCtx 创建实例；parentCtx 为 nil 则使用 rootCtx
// CreateWithContext uses parentCtx; if parentCtx is nil, uses rootCtx.
func (m *InstanceManager) CreateWithContext(
	parentCtx context.Context,
	typ, id string,
	cfg pluginapi.InstanceConfig,
) (pluginapi.Instance, error) {
	if typ == "" || id == "" {
		return nil, fmt.Errorf("empty typ or id")
	}

	f, ok := pluginapi.GetFactory(typ)
	if !ok {
		return nil, fmt.Errorf("no factory for type %q", typ)
	}

	inst, err := f.New(id, cfg)
	if err != nil {
		return nil, fmt.Errorf("create instance type=%s id=%s: %w", typ, id, err)
	}

	ctx := parentCtx
	if ctx == nil {
		ctx = m.rootCtx
	}

	// 关键：这里把 ctx + HostEnv 一起传给实例 Init
	// Key: inject ctx + HostEnv into instance.Init.
	if err := inst.Init(ctx, m.env); err != nil {
		return nil, fmt.Errorf("init instance type=%s id=%s: %w", typ, id, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[typ]; !ok {
		m.instances[typ] = make(map[string]pluginapi.Instance)
	}
	m.instances[typ][id] = inst

	return inst, nil
}

// Create 兼容简化接口：默认使用 rootCtx
// Create is a convenience wrapper using rootCtx.
func (m *InstanceManager) Create(
	typ, id string,
	cfg pluginapi.InstanceConfig,
) (pluginapi.Instance, error) {
	return m.CreateWithContext(nil, typ, id, cfg)
}

// Update / Destroy / DestroyAll / Get 保持和之前一样（只贴 Update 示例）
// The rest (Update / Destroy / DestroyAll / Get) remain unchanged.

func (m *InstanceManager) Update(
	typ, id string,
	cfg pluginapi.InstanceConfig,
) error {
	m.mu.RLock()
	byType, ok := m.instances[typ]
	if !ok {
		m.mu.RUnlock()
		return fmt.Errorf("no instances for type %q", typ)
	}
	inst, ok := byType[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no instance type=%s id=%s", typ, id)
	}
	return inst.UpdateConfig(cfg)
}

// DestroyAll 销毁所有实例，适合在进程退出时调用
// DestroyAll destroys all instances, suitable for process shutdown.
func (m *InstanceManager) DestroyAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for typ, byType := range m.instances {
		for id, inst := range byType {
			_ = inst.Close()
			delete(byType, id)
		}
		delete(m.instances, typ)
	}
}

// Get 获取某个实例
// Get returns an instance by type and ID.
func (m *InstanceManager) Get(typ, id string) (pluginapi.Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	byType, ok := m.instances[typ]
	if !ok {
		return nil, false
	}
	inst, ok := byType[id]
	return inst, ok
}

// Destroy 销毁单个实例（调用 Close 并从管理器中删除）
// Destroy destroys a single instance (calls Close and removes it).
func (m *InstanceManager) Destroy(typ, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	byType, ok := m.instances[typ]
	if !ok {
		return fmt.Errorf("no instances for type %q", typ)
	}
	inst, ok := byType[id]
	if !ok {
		return fmt.Errorf("no instance type=%s id=%s", typ, id)
	}

	if err := inst.Close(); err != nil {
		return fmt.Errorf("close instance type=%s id=%s: %w", typ, id, err)
	}

	delete(byType, id)
	if len(byType) == 0 {
		delete(m.instances, typ)
	}

	return nil
}
