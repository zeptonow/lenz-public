package assets

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
	GroupCache                  string            `mapstructure:"groupCache"`
	TopicCache                  string            `mapstructure:"topicCache"`
	AssetsOrigin                string            `mapstructure:"assetsOrigin"`
	AssetsSizeLimit             int               `mapstructure:"assetsSizeLimit"`
	AssetsRequestHeaders        map[string]string `mapstructure:"assetsRequestHeaders"`
	UseProfiler                 bool              `mapstructure:"useProfiler"`
	ClientKeyFilePath           string            `mapstructure:"clientKeyFilePath"`
	CaCertFilePath              string            `mapstructure:"caCertFilePath"`
	ClientCertFilePath          string            `mapstructure:"clientCertFilePath"`
}

func New(log logger.Logger) *Config {
	cfg := &Config{}
	configurator.Process(log, cfg)
	return cfg
}
