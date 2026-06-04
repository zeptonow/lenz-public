package http

import (
	"openreplay/backend/internal/config/common"
	"openreplay/backend/internal/config/configurator"
	"openreplay/backend/internal/config/objectstorage"
	"openreplay/backend/internal/config/redis"
	"openreplay/backend/pkg/env"
	"openreplay/backend/pkg/logger"
	"time"
)

type Config struct {
	common.Config               `mapstructure:"common"`
	common.Postgres             `mapstructure:"postgres"`
	redis.Redis                 `mapstructure:"redis"`
	objectstorage.ObjectsConfig `mapstructure:"s3"`
	common.HTTP                 `mapstructure:"http"`
	TopicRawWeb                 string `mapstructure:"topicRawWeb"`
	TopicRawMobile              string `mapstructure:"topicRawMobile"`
	common.Kafka                `mapstructure:",squash"`
	TopicRawImages              string        `mapstructure:"topicRawImages"`
	TopicCanvasImages           string        `mapstructure:"topicCanvasImages"`
	TopicRawAnalytics           string        `mapstructure:"topicRawAnalytics"`
	BeaconSizeLimit             int64         `mapstructure:"beaconSizeLimit"`
	CompressionThreshold        int64         `mapstructure:"compressionThreshold"`
	FileSizeLimit               int64         `mapstructure:"fileSizeLimit"`
	TokenSecret                 string        `mapstructure:"tokenSecret"`
	UAParserFile                string        `mapstructure:"uaParserFile"`
	MaxMinDBFile                string        `mapstructure:"maxmindDbFile"`
	UseProfiler                 bool          `mapstructure:"useProfiler"`
	ProjectExpiration           time.Duration `mapstructure:"projectExpiration"`
	RecordCanvas                bool          `mapstructure:"recordCanvas"`
	CanvasQuality               string        `mapstructure:"canvasQuality"`
	CanvasFps                   int           `mapstructure:"canvasFps"`
	MobileQuality               string        `mapstructure:"mobileQuality"`
	MobileFps                   int           `mapstructure:"mobileFps"`
	FSDir                       string        `mapstructure:"fsDir"`
	WorkerID                    uint16
}

func New(log logger.Logger) *Config {
	cfg := &Config{WorkerID: env.WorkerID()}
	configurator.Process(log, cfg)
	return cfg
}

// GetFSDir returns the filesystem directory path
func (c *Config) GetFSDir() string {
	return c.FSDir
}

// GetBucketName returns the S3 bucket name
func (c *Config) GetBucketName() string {
	return c.BucketName
}
