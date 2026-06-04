package kafka

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"openreplay/backend/pkg/logger"
)

// KafkaProducer is a confluent-kafka-go (librdkafka) producer with sticky
// keyed partitioning. The fnv1a_random partitioner is used so the
// (sessionID → partition) mapping matches segmentio/kafka-go's default
// `Hash{}` balancer (which also uses FNV-1a). This preserves session
// stickiness during the rollout from segmentio to confluent.
type KafkaProducer struct {
	producer *kafka.Producer
	logger   logger.Logger
	config   ProducerConfig
}

const produceMaxAttempts = 4 // 1 initial + 3 retries, matches the prior segmentio behavior.

func (kp *KafkaProducer) Produce(topic string, key uint64, value []byte) error {
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, key)
	return kp.produceWithRetry(topic, kafka.PartitionAny, keyBytes, value, key, -1)
}

// ProduceToPartition pins to a partition explicitly, bypassing the
// fnv1a_random partitioner. Use sparingly — only when the partition is part
// of the contract (e.g. dispatched by upstream).
func (kp *KafkaProducer) ProduceToPartition(topic string, partition, key uint64, value []byte) error {
	keyBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(keyBytes, key)
	return kp.produceWithRetry(topic, int32(partition), keyBytes, value, key, int64(partition))
}

func (kp *KafkaProducer) produceWithRetry(topic string, partition int32, key, value []byte, sessionID uint64, partitionForLog int64) error {
	t := topic
	build := func() *kafka.Message {
		return &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &t, Partition: partition},
			Key:            key,
			Value:          value,
		}
	}

	var lastErr error
	for attempt := 0; attempt < produceMaxAttempts; attempt++ {
		delivery := make(chan kafka.Event, 1)
		err := kp.producer.Produce(build(), delivery)
		if err == nil {
			ev := <-delivery
			m, ok := ev.(*kafka.Message)
			if ok && m.TopicPartition.Error == nil {
				return nil
			}
			if ok {
				err = m.TopicPartition.Error
			} else {
				err = fmt.Errorf("unexpected delivery event: %T", ev)
			}
		}
		lastErr = err
		if attempt < produceMaxAttempts-1 {
			if partitionForLog >= 0 {
				kp.logger.Warn(context.Background(),
					"Kafka publish to partition failed, retrying (attempt %d/%d): key=%d, partition=%d, error=%v",
					attempt+1, produceMaxAttempts-1, sessionID, partitionForLog, err,
				)
			} else {
				kp.logger.Warn(context.Background(),
					"Kafka publish failed, retrying (attempt %d/%d): key=%d, error=%v",
					attempt+1, produceMaxAttempts-1, sessionID, err,
				)
			}
		}
	}
	return lastErr
}

// Flush blocks up to `timeout` seconds while pending messages drain.
func (kp *KafkaProducer) Flush(timeout int) {
	if kp.producer == nil {
		return
	}
	kp.producer.Flush(timeout * 1000) // confluent expects ms.
}

func (kp *KafkaProducer) Close(timeout int) {
	if kp.producer == nil {
		return
	}
	kp.Flush(timeout)
	kp.producer.Close()
}

// NewKafkaProducer builds a confluent-kafka-go producer.
func NewKafkaProducer(
	log logger.Logger,
	brokers []string,
	config ProducerConfig,
) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka: brokers is empty")
	}

	cfg, err := buildProducerConfig(JoinBootstrapServers(brokers), config)
	if err != nil {
		return nil, err
	}

	p, err := kafka.NewProducer(cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka.NewProducer: %w", err)
	}

	return &KafkaProducer{
		producer: p,
		logger:   log,
		config:   config,
	}, nil
}

// buildProducerConfig is split out so it's unit-testable without a broker.
func buildProducerConfig(bootstrapServers string, config ProducerConfig) (*kafka.ConfigMap, error) {
	var acks string
	switch config.RequiredAcks {
	case 0:
		acks = "0"
	case 1:
		acks = "1"
	default:
		acks = "all"
	}

	extra := map[string]any{
		"acks":                acks,
		"partitioner":         "fnv1a_random", // matches segmentio/kafka-go Hash{} (FNV-1a)
		"linger.ms":           10,
		"go.delivery.reports": true,
	}
	if config.BatchSize > 0 {
		extra["batch.num.messages"] = config.BatchSize
	}
	if config.MaxAttempts > 0 {
		extra["message.send.max.retries"] = config.MaxAttempts
	}
	if config.CompressionType != "" {
		extra["compression.type"] = config.CompressionType
	}
	return BuildConfigMap(bootstrapServers, config.ClientConfig, extra)
}

// ProducerConfig holds Kafka producer configuration.
type ProducerConfig struct {
	BatchSize       int    // Number of messages to batch before sending (batch.num.messages)
	CompressionType string // "gzip", "snappy", "lz4", "zstd"
	MaxAttempts     int    // Maximum retry attempts (message.send.max.retries)
	RequiredAcks    int    // 0=none, 1=one, 2+=all
	ClientConfig    ClientConfig
}
