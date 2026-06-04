package redis

import "time"

type Redis struct {
	ConnectionURL     string        `mapstructure:"connectionUrl"`
	MaxLength         int64         `mapstructure:"maxLength"`
	ReadCount         int64         `mapstructure:"readCount"`
	ReadBlockDuration time.Duration `mapstructure:"readBlockDuration"`
	CloseTimeout      time.Duration `mapstructure:"closeTimeout"`
	UseRedisCache     bool          `mapstructure:"useCache"`
}
