package cmbus

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"sync"
	"time"

	"github.com/fluxionwatt/gridbeat/internal/models"
	"github.com/fluxionwatt/gridbeat/pluginapi"
	"github.com/fluxionwatt/gridbeat/utils/modbus"
	"github.com/sirupsen/logrus"
)

// ModbusConfig：单个 modbus 实例的配置
// ModbusConfig: configuration for a single modbus instance.
type InstanceConfig struct {
	Model models.Channel
	URL   string `mapstructure:"url"`
}

// ModbusInstance：具体插件实例，实现 pluginapi.Instance
// ModbusInstance: concrete plugin instance implementing pluginapi.Instance.
type ModbusInstance struct {
	id  string
	typ string

	cfg    InstanceConfig
	Status models.ChannelStatus

	logger  logrus.FieldLogger // 实例级 logger / per-instance logger
	server1 *modbus.ModbusRtuServer
	server2 *modbus.ModbusServer

	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	env *pluginapi.HostEnv

	mu   sync.Mutex
	init bool

	// this lock is used to avoid concurrency issues between goroutines, as
	// handler methods are called from different goroutines
	// (1 goroutine per client)
	lock sync.RWMutex

	// simple uptime counter, incremented in the main() above and exposed
	// as a 32-bit input register (2 consecutive 16-bit modbus registers).
	uptime uint32

	// these are here to hold client-provided (written) values, for both coils and
	// holding registers
	coils [100]bool

	inputReg   [65636]uint16
	holdingReg [65636]uint16

	// unix timestamp register, incremented in the main() function above and exposed
	// as a 32-bit holding register (2 consecutive 16-bit modbus registers).
	clock uint32
}

func (m *ModbusInstance) ID() string   { return m.id }
func (m *ModbusInstance) Type() string { return m.typ }

// ToStdLogger converts logrus.FieldLogger into *log.Logger in one call.
// It uses logrus's built-in WriterLevel (so formatter/hooks apply).
// IMPORTANT: call the returned closer.Close() when you no longer need the logger.
func ToStdLogger(l logrus.FieldLogger, level logrus.Level, prefix string, flags int) (*stdlog.Logger, io.Closer, error) {
	if l == nil {
		return nil, nil, fmt.Errorf("nil FieldLogger")
	}

	type writerLeveler interface {
		WriterLevel(level logrus.Level) *io.PipeWriter
	}
	wl, ok := any(l).(writerLeveler)
	if !ok {
		return nil, nil, fmt.Errorf("FieldLogger %T does not support WriterLevel (need *logrus.Logger or *logrus.Entry)", l)
	}

	pw := wl.WriterLevel(level) // goes through formatter + hooks
	return stdlog.New(pw, prefix, flags), pw, nil
}

// Init：用 parent ctx + HostEnv 初始化实例并启动轮询协程
// Init: initialize instance with parent ctx + HostEnv and start poller goroutine.
func (m *ModbusInstance) Init(parent context.Context, env *pluginapi.HostEnv) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.init {
		// 已经初始化过，通常不需要重复 Init
		// Already initialized; usually no need to re-init.
		return nil
	}

	if parent == nil {
		parent = context.Background()
	}
	m.parentCtx = parent
	m.env = env

	// 默认配置 / default config
	m.cfg.URL = "tcp://" + m.cfg.Model.TCPIPAddr + ":" + fmt.Sprintf("%d", m.cfg.Model.TCPPort)
	if m.cfg.Model.PhysicalLink == "serial" {
		m.cfg.URL = "rtu://" + m.cfg.Model.Device2
	}

	// logger：优先用 HostEnv.Logger，否则新建一个
	// logger: prefer HostEnv.Logger, otherwise create a new one.
	if env != nil && env.Logger != nil {
		m.logger = env.PluginLog.WithField("plugin", "cmbus").WithField("instance", m.id)
	}
	// 实例级 ctx / instance-level ctx
	m.ctx, m.cancel = context.WithCancel(parent)

	// 创建 Modbus server Init 都基于当前 cfg 创建一个新 server
	// Create Modbus server based on current cfg.

	var parity string
	if m.cfg.Model.Parity == 0 {
		parity = "N"
	} else {
		parity = "Y"
	}

	l, _, _ := ToStdLogger(m.logger, logrus.InfoLevel, "", 0)

	var err error
	if m.cfg.Model.PhysicalLink == "serial" {
		// for an RTU (serial) device/bus
		if m.server1, err = modbus.NewRTUServer(&modbus.ModbusRtuServerConfig{
			TTYPath:       m.cfg.Model.Device2,
			BaudRate:      m.cfg.Model.Speed,
			ModbusAddress: 0,
			DataBits:      m.cfg.Model.DataBits,
			StopBits:      m.cfg.Model.StopBits,
			Parity:        parity,
			Logger:        l,
		}, m); err != nil {
			return fmt.Errorf("modbus[%s]: create RTU server(%s) failed: %w", m.cfg.URL, m.id, err)
		}
	} else {
		if m.server2, err = modbus.NewServer(&modbus.ServerConfiguration{
			URL:     m.cfg.URL,
			Timeout: m.cfg.Model.OnnectTimeout,
			Logger:  l,
		}, m); err != nil {
			return fmt.Errorf("modbus[%s]: create TCP server(%s) failed: %w", m.cfg.URL, m.id, err)
		}
	}

	// 启动一个协程：负责自动 Open/Close + 周期读寄存器
	// Start one goroutine: handles Open/Close + periodic register reads.
	m.wg.Add(1)
	go func(cfg InstanceConfig) {
		defer m.wg.Done()
		m.runPoller(cfg)
	}(m.cfg)

	m.init = true
	m.logger.Infof("modbus simulator instance initialized, url=%s", m.cfg.URL)

	return nil
}

// runPoller：内部轮询逻辑，在单独协程中运行
// runPoller: internal polling loop, runs in a dedicated goroutine.
func (m *ModbusInstance) runPoller(cfg InstanceConfig) {

	m.logger.Infof("modbus simulator poller started, interval=%s", cfg.Model.RetryInterval)

	for {
		// 如果上层 ctx 已取消，直接退出
		// If parent context is done, exit.
		select {
		case <-m.ctx.Done():
			m.logger.Infof("modbus simulator poller exit: ctx=%v", m.ctx.Err())
			return
		default:
		}

		m.Status.Working = true
		m.Status.Linking = false

		// 尝试建立连接 / try to open connection.
		if err := m.server1.Start(); err != nil {
			m.logger.Errorf("modbus simulator open %s failed: %v", m.cfg.URL, err)
			if !sleepWithContext(m.ctx, 2*time.Second) {
				m.logger.Infof("modbus simulator poller exit during reconnect wait")
				return
			}
			continue
		}

		m.logger.Infof("modbus simulator connected to %s", cfg.URL)

		m.Status.Linking = true

		ticker := time.NewTicker(cfg.Model.RetryInterval)
		connected := true

		for connected {
			select {
			case <-m.ctx.Done():
				// 上层取消：关闭连接并退出
				// Parent canceled: close connection and exit.
				ticker.Stop()
				_ = m.server1.Stop()
				m.logger.Infof("modbus simulator poller exit on ctx done")
				return
			case <-ticker.C:
				// 轮询读寄存器 / poll holding/input registers.

				//m.cfg.Model.BytesReceived = m.cfg.Model.BytesReceived + 1
				//m.cfg.Model.BytesSent = m.cfg.Model.BytesSent + 1

				// 打印调试信息 / log debug values.
				m.logger.Debugf("modbus simulator recv ok")
			}
		}
	}
}

func (m *ModbusInstance) Get() any {
	return m.Status
}

// Close: stop poller and close client.
func (m *ModbusInstance) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.init {
		return nil
	}

	if m.cancel != nil {
		m.cancel()
	}

	// 等待轮询协程退出 / wait for poller goroutine to exit.
	m.wg.Wait()

	// 关闭底层 client（如果轮询协程已经关闭，这里 Close() 基本是幂等的）
	// Close underlying client (poller already closed it, so this is mostly idempotent).
	if m.server1 != nil {
		_ = m.server1.Stop()
		m.server1 = nil
	}

	m.ctx = nil
	m.cancel = nil
	m.init = false

	if m.logger != nil {
		m.logger.Infof("modbus simulator instance closed")
	}
	return nil
}

// UpdateConfig：支持运行时配置更新（包括 URL、端口等），必要时重启 Modbus 客户端
// UpdateConfig: support runtime config updates (including URL/port), restarting client if needed.
func (m *ModbusInstance) UpdateConfig(raw pluginapi.InstanceConfig) error {
	m.mu.Lock()

	newCfg := m.cfg

	if raw != nil {

		if v, ok := raw.(InstanceConfig); ok {
			newCfg = v
		}
	}

	// 填充默认值（防止被设置成 0） / fill defaults.

	// 判断是否需要重启（任意字段变化就重启，简单粗暴但安全）
	// Decide if restart is needed (restart on any change: simple and safe).
	needRestart := (newCfg != m.cfg)

	// 更新内存中的配置 / update in-memory cfg.
	m.cfg = newCfg

	parent := m.parentCtx
	env := m.env
	logger := m.logger

	m.mu.Unlock()

	if !needRestart {
		if logger != nil {
			logger.Infof("modbus simulator config updated without restart: url=%s", m.cfg.URL)
		}
		return nil
	}

	if logger != nil {
		logger.Infof("modbus simulator config changed, restarting client: url=%s", m.cfg.URL)
	}

	// 1) 关闭当前实例（停止轮询 + 关闭 client）
	// 1) Close current instance (stop poller and close client).
	if err := m.Close(); err != nil {
		return fmt.Errorf("modbus[%s] simulator: close before restart failed: %w", m.id, err)
	}

	// 2) 检查 parent ctx 是否还有效
	// 2) Ensure parent ctx is still valid.
	if parent == nil {
		parent = context.Background()
	}
	if err := parent.Err(); err != nil {
		if logger != nil {
			logger.Warnf("simulator parent context canceled, skip restart: %v", err)
		}
		return err
	}

	// 3) 用新配置 + 原来的 parent/env 重新 Init
	// 3) Re-init with new cfg + original parent/env.
	return m.Init(parent, env)
}

// ModbusFactory：实现 Factory 接口
// ModbusFactory: implements pluginapi.Factory.
type ModbusFactory struct{}

func (f *ModbusFactory) Type() string { return "cmbus" }

// New：根据配置创建实例（真正启动在 Init 中完成）
// New: create an instance from config (real start happens in Init).
func (f *ModbusFactory) New(id string, raw pluginapi.InstanceConfig) (pluginapi.Instance, error) {
	if id == "" {
		return nil, fmt.Errorf("modbus simulator: empty instance id")
	}

	var cfg InstanceConfig

	if raw != nil {
		if v, ok := raw.(InstanceConfig); ok {
			cfg = v
		}
	}

	return &ModbusInstance{
		id:  id,
		typ: f.Type(),
		cfg: cfg,
	}, nil
}

// init：注册工厂
// init: register factory.
func init() {
	pluginapi.RegisterFactory(&ModbusFactory{})
}

// sleepWithContext：带 ctx 的 sleep，返回是否正常 sleep 完成
// sleepWithContext: sleep with ctx, returns whether it completed normally.
func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
