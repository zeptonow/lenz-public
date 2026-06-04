package redisstream

import (
	"regexp"

	"github.com/docker/distribution/context"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// RedisConfig holds the connection config for the redis stream client
type RedisConfig struct {
	ConnectionURL string
}

var redisStreamCfg *RedisConfig

// SetConfig sets the redis connection config for the stream module.
// Must be called before NewProducer or any getRedisClient call.
func SetConfig(cfg *RedisConfig) {
	redisStreamCfg = cfg
}

func getRedisClient() (*redis.Client, error) {
	if redisClient != nil {
		return redisClient, nil
	}

	connectionString := ""
	if redisStreamCfg != nil {
		connectionString = redisStreamCfg.ConnectionURL
	}
	if connectionString == "" {
		// Fallback: try env var for backwards compat
		connectionString = "redis://localhost:6379"
	}

	match, _ := regexp.MatchString("^[^:]+://", connectionString)
	if !match {
		connectionString = "redis://" + connectionString
	}

	options, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}

	redisClient = redis.NewClient(options)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return redisClient, nil
}
