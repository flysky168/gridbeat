package goose

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fluxionwatt/gridbeat/pluginapi"
)

// GooseConfig：单个 goose 实例的配置
// GooseConfig holds configuration for a single goose instance.
type GooseConfig struct {
	DSN           string
	FlushInterval time.Duration
}

// GooseInstance：具体实例实现
// GooseInstance is a concrete plugin instance.
type GooseInstance struct {
	id  string
	typ string
	cfg GooseConfig

	logger logrus.FieldLogger // 实例级 logger / per-instance logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	init   bool
}

func (g *GooseInstance) ID() string   { return g.id }
func (g *GooseInstance) Type() string { return g.typ }

// Init：从 env 里拿到 logger + 用 parent ctx 派生实例 ctx
// Init: get logger from env + derive instance ctx from parent.
func (g *GooseInstance) Init(parent context.Context, env *pluginapi.HostEnv) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.init {
		return nil
	}
	if parent == nil {
		parent = context.Background()
	}

	// 配置默认值 / default config
	if g.cfg.FlushInterval == 0 {
		g.cfg.FlushInterval = 5 * time.Second
	}

	// 设置 logger：
	// - 如果 HostEnv 有 Logger，则在其基础上加字段
	// - 否则创建一个新的 logrus.Logger
	// Setup logger:
	// - If HostEnv has Logger, use it with fields
	// - Otherwise, create a new logger.
	if env != nil && env.Logger != nil {
		g.logger = env.PluginLog.WithField("plugin", "goose").WithField("instance", g.id)
	}

	// 实例级 ctx / instance-level ctx
	g.ctx, g.cancel = context.WithCancel(parent)

	g.logger.Infof("init instance, dsn=%s interval=%s", g.cfg.DSN, g.cfg.FlushInterval)

	// 启动多个协程，都监听 g.ctx.Done()
	// Start multiple goroutines, all select on g.ctx.Done().

	// goroutine A: flush loop
	g.wg.Add(1)
	go func(cfg GooseConfig) {
		defer g.wg.Done()
		ticker := time.NewTicker(cfg.FlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				opCtx, cancel := context.WithTimeout(g.ctx, 2*time.Second)
				_ = g.doFlush(opCtx)
				cancel()
			case <-g.ctx.Done():
				g.logger.Infof("flush goroutine exit, ctx=%v", g.ctx.Err())
				return
			}
		}
	}(g.cfg)

	// goroutine B: metrics loop
	g.wg.Add(1)
	go func(cfg GooseConfig) {
		defer g.wg.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				g.logger.Infof("metrics heartbeat, dsn=%s", cfg.DSN)
			case <-g.ctx.Done():
				g.logger.Infof("metrics goroutine exit, ctx=%v", g.ctx.Err())
				return
			}
		}
	}(g.cfg)

	g.init = true
	return nil
}

func (g *GooseInstance) doFlush(ctx context.Context) error {
	select {
	case <-time.After(500 * time.Millisecond):
		g.logger.Infof("doFlush done")
		return nil
	case <-ctx.Done():
		g.logger.Warnf("doFlush canceled: %v", ctx.Err())
		return ctx.Err()
	}
}

// Close：cancel ctx + 等待所有协程退出
// Close: cancel ctx + wait for all goroutines to exit.
func (g *GooseInstance) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.init {
		return nil
	}
	if g.cancel != nil {
		g.cancel()
	}
	g.wg.Wait()
	g.ctx = nil
	g.cancel = nil
	g.init = false

	g.logger.Infof("close instance")
	return nil
}

func (m *GooseInstance) Get() any {
	return nil
}

// UpdateConfig：更新配置（这里简单更新 cfg，不重启协程；你也可以选择 Close+Init 重启）
// UpdateConfig: update config (simple cfg update; you may choose Close+Init to restart).
func (g *GooseInstance) UpdateConfig(raw pluginapi.InstanceConfig) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	var newCfg GooseConfig
	if raw != nil {
		if v, ok := raw.(GooseConfig); ok {
			newCfg.DSN = v.DSN
			newCfg.FlushInterval = v.FlushInterval
		}
	}
	if newCfg.FlushInterval == 0 {
		newCfg.FlushInterval = 5 * time.Second
	}

	// 简单策略：只更新内存中的 cfg（新起的 goroutine 可读取新的配置）
	// Simple strategy: just update cfg in memory.
	g.cfg = newCfg
	g.logger.Infof("config updated: dsn=%s interval=%s", g.cfg.DSN, g.cfg.FlushInterval)
	return nil
}

// GooseFactory 同前，只是 New 不需要 env
// GooseFactory is unchanged: New does not receive env.
type GooseFactory struct{}

func (f *GooseFactory) Type() string { return "goose" }

func (f *GooseFactory) New(id string, raw pluginapi.InstanceConfig) (pluginapi.Instance, error) {
	if id == "" {
		return nil, fmt.Errorf("goose: empty instance id")
	}
	var cfg GooseConfig
	if raw != nil {
		if v, ok := raw.(GooseConfig); ok {
			cfg.DSN = v.DSN
			cfg.FlushInterval = v.FlushInterval
		}
	}
	return &GooseInstance{
		id:  id,
		typ: f.Type(),
		cfg: cfg,
	}, nil
}

func init() {
	pluginapi.RegisterFactory(&GooseFactory{})
}
