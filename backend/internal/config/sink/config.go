package sink

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
)

type Config struct {
	common.Config        `mapstructure:"common"`
	redis.Redis          `mapstructure:"redis"`
	FsDir                string `mapstructure:"fsDir"`
	FsUlimit             uint16 `mapstructure:"fsUlimit"`
	FileBuffer           int    `mapstructure:"fileBuffer"`
	SyncTimeout          int    `mapstructure:"syncTimeout"`
	GroupSink            string `mapstructure:"groupSink"`
	GroupSinkMobile      string `mapstructure:"groupSinkMobile"`
	TopicRawWeb          string `mapstructure:"topicRawWeb"`
	TopicRawMobile       string `mapstructure:"topicRawMobile"`
	common.Kafka         `mapstructure:",squash"`
	TopicCache           string `mapstructure:"topicCache"`
	TopicTrigger         string `mapstructure:"topicTrigger"`
	TopicMobileTrigger   string `mapstructure:"topicMobileTrigger"`
	CacheAssets          bool   `mapstructure:"cacheAssets"`
	AssetsOrigin         string `mapstructure:"assetsOrigin"`
	ProducerCloseTimeout int    `mapstructure:"producerCloseTimeout"`
	CacheThreshold       int64  `mapstructure:"cacheThreshold"`
	CacheExpiration      int64  `mapstructure:"cacheExpiration"`
	CacheBlackList       string `mapstructure:"cacheBlackList"`
	UseProfiler          bool   `mapstructure:"useProfiler"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
