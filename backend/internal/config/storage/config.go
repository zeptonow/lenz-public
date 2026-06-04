package storage

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/objectstorage"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
	"time"
)

type Config struct {
	common.Config               `mapstructure:"common"`
	redis.Redis                 `mapstructure:"redis"`
	objectstorage.ObjectsConfig `mapstructure:"s3"`
	FSDir                       string        `mapstructure:"fsDir"`
	FileSplitSize               int           `mapstructure:"fileSplitSize"`
	FileSplitTime               time.Duration `mapstructure:"fileSplitTime"`
	RetryTimeout                time.Duration `mapstructure:"retryTimeout"`
	GroupStorage                string        `mapstructure:"groupStorage"`
	common.Kafka                `mapstructure:",squash"`
	TopicTrigger                string        `mapstructure:"topicTrigger"`
	GroupFailover               string        `mapstructure:"groupFailover"`
	TopicFailover               string        `mapstructure:"topicFailover"`
	DeleteTimeout               time.Duration `mapstructure:"deleteTimeout"`
	ProducerCloseTimeout        int           `mapstructure:"producerCloseTimeout"`
	UseFailover                 bool          `mapstructure:"useFailover"`
	MaxFileSize                 int64         `mapstructure:"maxFileSize"`
	UseSort                     bool          `mapstructure:"useSort"`
	UseProfiler                 bool          `mapstructure:"useProfiler"`
	CompressionAlgo             string        `mapstructure:"compressionAlgo"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
