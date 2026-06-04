package canvases

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
	CanvasDir                   string `mapstructure:"canvasDir"`
	TopicCanvasImages           string `mapstructure:"topicCanvasImages"`
	TopicCanvasTrigger          string `mapstructure:"topicCanvasTrigger"`
	common.Kafka                `mapstructure:",squash"`
	GroupCanvasImage            string `mapstructure:"groupCanvasImage"`
	GroupCanvasTrigger          string `mapstructure:"groupCanvasTrigger"`
	UseProfiler                 bool   `mapstructure:"useProfiler"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
