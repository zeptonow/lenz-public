package redisstream

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Producer struct {
	redis        *redis.Client
	maxLenApprox int64
	ttl          time.Duration
}

// ProducerConfig holds config values for the redis stream producer
type ProducerConfig struct {
	MaxLength  int64
	TTLSeconds uint64
}

var producerCfg *ProducerConfig

// SetProducerConfig sets the config for the redis stream producer.
// Must be called before NewProducer.
func SetProducerConfig(cfg *ProducerConfig) {
	producerCfg = cfg
}

func NewProducer() *Producer {
	redClient, err := getRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	var ttlSeconds uint64 = 14400
	var maxLen int64 = 10000
	if producerCfg != nil {
		if producerCfg.TTLSeconds > 0 {
			ttlSeconds = producerCfg.TTLSeconds
		}
		if producerCfg.MaxLength > 0 {
			maxLen = producerCfg.MaxLength
		}
	}
	return &Producer{
		redis:        redClient,
		maxLenApprox: maxLen,
		ttl:          time.Duration(ttlSeconds) * time.Second,
	}
}

func (p *Producer) Produce(topic string, key uint64, value []byte) error {
	ctx := context.Background()

	args := &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"sessionID": key,
			"value":     value,
		},
	}

	if p.ttl > 0 {
		cutoffTime := time.Now().Add(-p.ttl).UnixMilli()
		args.MinID = fmt.Sprintf("%d-0", cutoffTime)
		args.Approx = true
	} else if p.maxLenApprox > 0 {
		args.MaxLen = p.maxLenApprox
		args.Approx = true
	}

	_, err := p.redis.XAdd(ctx, args).Result()
	if err != nil {
		return err
	}

	return nil
}

func (p *Producer) ProduceToPartition(topic string, partition, key uint64, value []byte) error {
	return nil
}

func (p *Producer) Close(_ int) {
	// noop
}

func (p *Producer) Flush(_ int) {
	// noop
}
