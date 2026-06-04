package integrations

import (
	"time"

	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/objectstorage"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/env"
	"openreplay/backend/pkg/logger"
)

type Config struct {
	common.Config               `mapstructure:"common"`
	common.Postgres             `mapstructure:"postgres"`
	redis.Redis                 `mapstructure:"redis"`
	objectstorage.ObjectsConfig `mapstructure:"s3"`
	common.HTTP                 `mapstructure:"integrationsHttp"`
	common.RateLimiter          `mapstructure:"rateLimiter"`
	ProjectExpiration           time.Duration `mapstructure:"projectExpiration"`
	WorkerID                    uint16
}

func New(log logger.Logger) *Config {
	cfg := &Config{WorkerID: env.WorkerID()}
	configurator.Process(log, cfg)
	return cfg
}
