package heuristics

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/pprof"
)

type Config struct {
	common.Config   `mapstructure:"common"`
	redis.Redis     `mapstructure:"redis"`
	GroupHeuristics string `mapstructure:"groupHeuristics"`
	TopicAnalytics  string `mapstructure:"topicAnalytics"`
	LoggerTimeout   int    `mapstructure:"loggerTimeout"`
	TopicRawWeb     string `mapstructure:"topicRawWeb"`
	TopicRawMobile  string `mapstructure:"topicRawMobile"`
	ProducerTimeout int    `mapstructure:"producerTimeout"`
	UseProfiler     bool   `mapstructure:"useProfiler"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	if cfg.UseProfiler {
		pprof.StartProfilingServer()
	}
	return cfg
}
