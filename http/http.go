package http

import (
	"io/fs"
	"strings"

	"github.com/fluxionwatt/gridbeat/frontend"
	"github.com/gofiber/contrib/v3/monitor"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/static"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/sirupsen/logrus"
)

func NewHandler(
	server *mqtt.Server, extra string, errorLogger *logrus.Logger, accessLogger *logrus.Logger,
) (*fiber.App, error) {

	app := fiber.New(fiber.Config{
		// 统一错误处理 + logrus
		ErrorHandler: func(c fiber.Ctx, err error) error {
			errorLogger.WithFields(logrus.Fields{
				"path":   c.Path(),
				"ip":     c.IP(),
				"method": c.Method(),
			}).WithError(err).Error("fiber error")

			// 自定义返回
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"code":  code,
				"error": err.Error(),
			})
		},
	})
	app.Use(func(c fiber.Ctx) error {
		c.Locals("logger", errorLogger)
		return c.Next()
	})

	app.Use(pprof.New(pprof.Config{Prefix: "/endpoint-prefix"}))
	app.Use(requestid.New())

	app.Use("/public/extra", static.New(extra, static.Config{
		Browse:    true,
		Download:  true,
		ByteRange: true,
		Compress:  true,
	}))

	//app.Get("/healthz", healthcheck.New())

	//cfg := swaggerui.Config{
	//	BasePath: "/",
	//		FilePath: "./docs/swagger.json",
	//		Path:     "swagger",
	//		Title:    "Swagger API Docs",
	//	}

	//app.Use(swaggerui.New(cfg))

	// JWT Middleware
	//app.Use(jwtware.New(jwtware.Config{
	//	SigningKey: jwtware.SigningKey{Key: []byte("secret")},
	//}))
	app.Use(AccessLogMiddleware(accessLogger))

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e interface{}) {
			errorLogger.WithFields(logrus.Fields{
				"path":   c.Path(),
				"ip":     c.IP(),
				"method": c.Method(),
			}).Errorf("fiber panic recovered: %v", e)
		},
	}))

	app.Get("/metrics", monitor.New(monitor.Config{Title: "MyService Metrics Page"}))

	app.Get("/v1/api/test", func(c fiber.Ctx) error {
		return c.SendString("I'm a GET request!")
	})

	// WebSocket 握手预处理中间件
	// 只允许 WebSocket 升级的请求进入后面的路由
	app.Use("/ws", func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket + JWT
	// 前端连接示例：
	//   ws://host/ws/ssh?host=192.168.1.10&port=22&user=root&token=eyJ...
	//app.Get("/ws/ssh", auth.JWTMiddleware(), websocket.New(terminal.SSHWebsocket))
	app.Get("/ws/ssh", websocket.New(SSHWebsocket))

	distFS, _ := fs.Sub(frontend.Assets(), "dist")
	app.Use("/", static.New("", static.Config{
		FS:            distFS,
		IndexNames:    []string{"index.html"},
		CacheDuration: -1,
		MaxAge:        0,
		ModifyResponse: func(c fiber.Ctx) error {
			// 顺手把浏览器缓存也干掉
			c.Response().Header.Del("Last-Modified")
			c.Response().Header.Del("Etag")
			c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
			return nil
		},
	}))

	return app, nil
}

func ForceHTTPS(httpsPort string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// 已经是 https，直接放行
		if strings.EqualFold(c.Protocol(), "https") {
			return c.Next()
		}

		// 反向代理场景：X-Forwarded-Proto 已经是 https 也放行
		if proto := c.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
			return c.Next()
		}

		// 普通 HTTP 请求 → 301 跳到 HTTPS
		host := c.Hostname()
		uri := c.OriginalURL()

		if httpsPort == "443" {
			// 生产：用默认 443，不拼端口
			// 如需严谨，可以把原 host 中的 :80 裁掉，这里先略
		} else {
			// 开发：HTTP :8080 → HTTPS :8443
			host = host + ":" + httpsPort
		}

		target := "https://" + host + uri

		// v3 正确写法：先取 Redirect()，再 To()
		c.Redirect().To(target)
		return nil
	}
}
