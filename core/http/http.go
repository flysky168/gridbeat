package http

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/fluxionwatt/gridbeat/core"
	"github.com/fluxionwatt/gridbeat/internal/api"
	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/sirupsen/logrus"
)

// HTTPServerConfig：单个 httpserver 实例的配置
// HTTPServerConfig: configuration for a single httpserver instance.
type Config struct {
	HTTPAddress  string
	HTTPSAddress string
	HTTPSDisable bool
}

// HTTPServerInstance：具体实例，实现 pluginapi.Instance
// HTTPServerInstance: concrete instance implementing pluginapi.Instance.
type HTTPServerInstance struct {
	api.Server

	cfg Config

	logger logrus.FieldLogger // 实例级 logger / per-instance logger
	app    *fiber.App

	//Conf         *config.Config
	AccessLogger *logrus.Logger

	// parentCtx：Init 传入的父 context，用于重启时复用
	// parentCtx: parent context passed to Init, reused on restart.
	parentCtx context.Context

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu   sync.Mutex
	init bool
}

// Init：使用 parent ctx + HostEnv 初始化实例，并启动 HTTP 服务
// Init: initialize instance with parent ctx + HostEnv, and start HTTP server.
func (s *HTTPServerInstance) Init(parent context.Context, cycle *core.Cycle) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.init {
		// 已经初始化过，通常不需要重复 Init
		// Already initialized; usually no need to re-init.
		return nil
	}

	if parent == nil {
		parent = context.Background()
	}
	s.parentCtx = parent

	// 设置 logger：优先用 HostEnv.Logger，其次自己创建
	// Setup logger: prefer HostEnv.Logger, otherwise create a new one.

	s.logger = cycle.Logger.WithField("plugin", "http")

	// 为该实例创建独立 ctx，用于控制 Fiber 和相关协程生命周期
	// Create an instance-level ctx, to control Fiber and related goroutines.
	s.ctx, s.cancel = context.WithCancel(parent)

	s.app = NewHandler(s.logger)

	s.app.Use(AccessLogMiddleware(cycle.AccessLogger))

	s.app.Use("/public/extra", static.New(core.Gconfig.ExtraPath, static.Config{
		Browse:    true,
		Download:  true,
		ByteRange: true,
		Compress:  true,
	}))

	s.app.Use("/public/log", static.New(core.Gconfig.LogPath, static.Config{
		Browse:    true,
		Download:  true,
		ByteRange: true,
		Compress:  true,
	}))

	// Swagger UI: http://localhost:8080/swagger/index.html
	s.app.Get("/swagger/*", swaggo.HandlerDefault)

	s.Server.Cfg = cycle.Conf
	s.Server.DB = cycle.DB
	s.Server.MQTT = cycle.MQTT
	s.Server.Mgr = cycle.Mgr

	s.Server.Route(s.app)

	s.wg.Add(1)
	go func(addr string) {
		defer s.wg.Done()

		s.logger.Infof("starting fiber HTTP server on %s", addr)

		// Listen 会阻塞直到 Shutdown 被调用或发生错误
		// Listen blocks until Shutdown is called or an error occurs.
		if err := s.app.Listen(addr, fiber.ListenConfig{
			DisableStartupMessage: true,
		}); err != nil {
			// Shutdown 后 Listen 通常会返回错误，可以按需过滤
			// After Shutdown, Listen usually returns an error; filter/log as needed.
			s.logger.Warnf("fiber.Listen returned: %v", err)
		}

		s.logger.Infof("Fiber HTTP server stopped (addr=%s)", addr)

	}(s.cfg.HTTPAddress)

	// 协程 1：启动 Fiber HTTP 服务器（阻塞 Listen）
	// Goroutine 1: start Fiber HTTP server (blocking Listen).
	s.wg.Add(1)
	go func(addr string) {
		defer s.wg.Done()

		s.logger.Infof("starting fiber HTTPS server on %s", addr)

		cert, _ := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))

		tlsConf := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		ln, err := tls.Listen("tcp", addr, tlsConf)
		if err != nil {
			s.logger.Fatal("tls listen failed: ", err)
		}

		if err := s.app.Listener(ln, fiber.ListenConfig{
			DisableStartupMessage: true,
			TLSMinVersion:         tls.VersionTLS12,
		}); err != nil {
			s.logger.Fatal(err)
		}

		s.logger.Infof("Fiber HTTPS server stopped (addr=%s)", addr)

	}(s.cfg.HTTPSAddress)

	// 协程 2：监听 ctx.Done()，触发 Fiber 优雅关闭
	// Goroutine 2: watch ctx.Done() and trigger graceful Fiber shutdown.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		<-s.ctx.Done()
		s.logger.Infof("context canceled, shutting down Fiber HTTP server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		doneCh := make(chan struct{})
		go func() {
			if err := s.app.Shutdown(); err != nil {
				s.logger.Errorf("Fiber shutdown error: %v", err)
			}
			close(doneCh)
		}()

		select {
		case <-doneCh:
			s.logger.Infof("Fiber HTTP server shutdown completed")
		case <-shutdownCtx.Done():
			s.logger.Warnf("Fiber HTTP server shutdown timed out: %v", shutdownCtx.Err())
		}
	}()

	s.init = true
	return nil
}

// Close：取消实例 ctx，等待所有协程退出
// Close: cancel instance ctx and wait for all goroutines to exit.
func (s *HTTPServerInstance) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.init {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	// 等待监听协程 + 关闭协程全部退出
	// Wait for listener goroutine and shutdown watcher to exit.
	s.wg.Wait()

	s.ctx = nil
	s.cancel = nil
	s.app = nil
	s.init = false

	if s.logger != nil {
		s.logger.Infof("httpserver instance closed")
	}
	return nil
}

// New：根据配置创建实例（不启动服务，Init 时才启动）
// New: create an instance from config (server starts in Init).
func New(cfg Config) *HTTPServerInstance {

	return &HTTPServerInstance{
		cfg: cfg,
	}
}
