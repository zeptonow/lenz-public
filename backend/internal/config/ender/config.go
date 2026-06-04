package ender

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
	"time"
)

type Config struct {
	common.Config     `mapstructure:"common"`
	common.Postgres   `mapstructure:"postgres"`
	redis.Redis       `mapstructure:"redis"`
	ProjectExpiration time.Duration `mapstructure:"projectExpiration"`
	GroupEnder        string        `mapstructure:"groupEnder"`
	GroupEnderMobile  string        `mapstructure:"groupEnderMobile"`
	LoggerTimeout     int           `mapstructure:"loggerTimeout"`
	TopicRawWeb       string        `mapstructure:"topicRawWeb"`
	TopicRawMobile    string        `mapstructure:"topicRawMobile"`
	TopicCanvasImages string        `mapstructure:"topicCanvasImages"`
	TopicRawImages    string        `mapstructure:"topicRawImages"`
	common.Kafka      `mapstructure:",squash"`
	ProducerTimeout   int  `mapstructure:"producerTimeout"`
	PartitionsNumber  int  `mapstructure:"partitionsNumber"`
	UseEncryption     bool `mapstructure:"useEncryption"`
	UseProfiler       bool `mapstructure:"useProfiler"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
