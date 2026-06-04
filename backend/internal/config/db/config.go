package db

import (
	"time"

	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/logger"
)

type Config struct {
	common.Config         `mapstructure:"common"`
	common.Postgres       `mapstructure:"postgres"`
	common.Clickhouse     `mapstructure:"clickhouse"`
	redis.Redis           `mapstructure:"redis"`
	ProjectExpiration     time.Duration `mapstructure:"projectExpiration"`
	LoggerTimeout         int           `mapstructure:"loggerTimeout"`
	GroupDB               string        `mapstructure:"groupDb"`
	GroupAnalytics        string        `mapstructure:"groupAnalytics"`
	TopicRawWeb           string        `mapstructure:"topicRawWeb"`
	TopicAnalytics        string        `mapstructure:"topicAnalytics"`
	TopicRawMobile        string        `mapstructure:"topicRawMobile"`
	TopicRawAnalytics     string        `mapstructure:"topicRawAnalytics"`
	CommitBatchTimeout    time.Duration `mapstructure:"commitBatchTimeout"`
	BatchQueueLimit       int           `mapstructure:"batchQueueLimit"`
	BatchSizeLimit        int           `mapstructure:"batchSizeLimit"`
	UseProfiler           bool          `mapstructure:"useProfiler"`
	PAUpdaterStartTime    string        `mapstructure:"paUpdaterStartTime"`
	PAUpdaterEndTime      string        `mapstructure:"paUpdaterEndTime"`
	PAUpdaterTickDuration time.Duration `mapstructure:"paUpdaterTickDuration"`
	CHReadBatchSizeLimit  int           `mapstructure:"chReadBatchSizeLimit"`
	CHSendBatchSizeLimit  int           `mapstructure:"chSendBatchSizeLimit"`
	CHReadUsersSizeLimit  int           `mapstructure:"chReadUsersSizeLimit"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
