package api

import (
	"time"

	"github.com/fluxionwatt/gridbeat/version"
	"github.com/gofiber/fiber/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func (s *Server) Version(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"productName": version.ProductName,
		"Version":     version.Version,
		"buildTime":   version.BUILDTIME,
		"gitCommit":   version.CommitSHA,
		"copyright":   "© 2025 Your Company. All rights reserved.",
		"extra":       "版本信息",
	})
}

func (s *Server) MaintenanceOverview(c fiber.Ctx) error {

	//log := GetLogger(c)

	up, _ := host.Uptime()

	_, _ = cpu.Percent(time.Second, false)
	percentages, _ := cpu.Percent(time.Second, false)
	cpuUsed := percentages[0]

	v, err := mem.VirtualMemory()
	if err != nil {
		return fiber.ErrBadRequest
	}

	data := fiber.Map{
		"uptimeSeconds": up,
		"cpuUsage":      cpuUsed,
		"memUsage":      v.UsedPercent,
		"services": []fiber.Map{
			fiber.Map{"name": "Modbus", "status": "up"},
			fiber.Map{"name": "MQTT", "status": "down"},
			fiber.Map{"name": "Northbound", "status": "degraded"},
		},
	}
	return c.JSON(data)
}
