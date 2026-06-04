package api

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
	common.Clickhouse           `mapstructure:"clickhouse"`
	redis.Redis                 `mapstructure:"redis"`
	objectstorage.ObjectsConfig `mapstructure:"s3"`
	common.HTTP                 `mapstructure:"apiHttp"`
	common.RateLimiter          `mapstructure:"rateLimiter"`
	FSDir                       string        `mapstructure:"fsDir"`
	SpotsDir                    string        `mapstructure:"spotsDir"`
	ProjectExpiration           time.Duration `mapstructure:"projectExpiration"`
	MinimumStreamDuration       int           `mapstructure:"minimumStreamDuration"`
	AssistUrl                   string        `mapstructure:"assistUrl"`
	AssistKey                   string        `mapstructure:"assistKey"`
	AssistLiveSuffix            string        `mapstructure:"assistLiveSuffix"`
	AssistListSuffix            string        `mapstructure:"assistListSuffix"`
	AssistRequestTimeout        time.Duration `mapstructure:"assistRequestTimeout"`
	AssistJwtSecret             string        `mapstructure:"assistJwtSecret"`
	AssistJwtAlgorithm          string        `mapstructure:"assistJwtAlgorithm"`
	AssistJwtIssuer             string        `mapstructure:"assistJwtIssuer"`
	AssistJwtExpiration         int64         `mapstructure:"assistJwtExpiration"`
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
