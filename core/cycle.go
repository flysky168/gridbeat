package core

import (
	"sync"

	"github.com/fluxionwatt/gridbeat/internal/config"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Cycle struct {
	Conf         *config.Config
	DB           *gorm.DB
	Logger       logrus.FieldLogger
	MQTT         *mqtt.Server
	Mgr          *InstanceManager
	AccessLogger *logrus.Logger
	WG           *sync.WaitGroup
}
