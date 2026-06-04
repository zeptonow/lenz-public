package images

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/objectstorage"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
)

type Config struct {
	common.Config               `mapstructure:"common"`
	redis.Redis                 `mapstructure:"redis"`
	objectstorage.ObjectsConfig `mapstructure:"s3"`
	FSDir                       string `mapstructure:"fsDir"`
	ScreenshotsDir              string `mapstructure:"screenshotsDir"`
	TopicRawImages              string `mapstructure:"topicRawImages"`
	common.Kafka                `mapstructure:",squash"`
	GroupImageStorage           string `mapstructure:"groupImageStorage"`
	UseProfiler                 bool   `mapstructure:"useProfiler"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
