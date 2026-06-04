package queue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/queue/kafka"
	"openreplay/backend/pkg/queue/types"
	"openreplay/backend/pkg/redisstream"
)

// KafkaConn is bootstrap + SASL settings for one Kafka client role (producer or consumer).
type KafkaConn struct {
	Brokers          []string
	BootstrapServers string
	SASLUsername     string
	SASLPassword     string
}

// KafkaSettings holds producer and consumer connection settings (may differ).
type KafkaSettings struct {
	ProducerBrokers      string
	ProducerSaslUsername string
	ProducerSaslPassword string
	ConsumerBrokers      string
	ConsumerSaslUsername string
	ConsumerSaslPassword string
	ConsumerPartition    *int
}

// RawKafkaOpts enables Kafka for configured queue topic names.
// Producer and Consumer may use different clusters/credentials.
// Partitions is for Kafka consumers only; set via SetConsumerPartitions before NewRawTopicConsumer.
type RawKafkaOpts struct {
	Producer           KafkaConn
	Consumer           KafkaConn
	ConsumerPartition  *int
	TopicRawWeb        string
	TopicRawMobile     string
	TopicTrigger       string
	TopicMobileTrigger string
	TopicRawImages     string
	TopicCanvasImages  string
	TopicCanvasTrigger string
	Partitions         []int
}

func (o *RawKafkaOpts) ProducerEnabled() bool {
	return o != nil && len(o.Producer.Brokers) > 0
}

func (o *RawKafkaOpts) ConsumerEnabled() bool {
	return o != nil && len(o.Consumer.Brokers) > 0
}

// ParseBrokerCSV splits comma-separated broker addresses (host:port only).
func ParseBrokerCSV(s string) []string {
	return kafka.ParseBrokerCSV(s)
}

func parseKafkaConn(brokersCSV, saslUsername, saslPassword string) KafkaConn {
	brokers := ParseBrokerCSV(brokersCSV)
	if len(brokers) == 0 {
		return KafkaConn{}
	}
	return KafkaConn{
		Brokers:          brokers,
		BootstrapServers: kafka.JoinBootstrapServers(brokers),
		SASLUsername:     strings.TrimSpace(saslUsername),
		SASLPassword:     saslPassword,
	}
}

// NewRawKafkaOpts builds RawKafkaOpts from producer/consumer settings.
// Returns an error when neither producer nor consumer brokers are configured.
func NewRawKafkaOpts(settings KafkaSettings, fill func(*RawKafkaOpts)) (*RawKafkaOpts, error) {
	producer := parseKafkaConn(settings.ProducerBrokers, settings.ProducerSaslUsername, settings.ProducerSaslPassword)
	consumer := parseKafkaConn(settings.ConsumerBrokers, settings.ConsumerSaslUsername, settings.ConsumerSaslPassword)
	if len(producer.Brokers) == 0 && len(consumer.Brokers) == 0 {
		return nil, fmt.Errorf("kafka: no producer or consumer brokers configured")
	}
	o := &RawKafkaOpts{
		Producer:          producer,
		Consumer:          consumer,
		ConsumerPartition: settings.ConsumerPartition,
	}
	if fill != nil {
		fill(o)
	}
	return o, nil
}

// RawKafkaOptsFromBrokers is deprecated; use NewRawKafkaOpts with the same values for producer and consumer.
func RawKafkaOptsFromBrokers(
	brokersCSV, saslUsername, saslPassword string,
	consumerPartition *int,
	fill func(*RawKafkaOpts),
) (*RawKafkaOpts, error) {
	return NewRawKafkaOpts(KafkaSettings{
		ProducerBrokers:      brokersCSV,
		ProducerSaslUsername: saslUsername,
		ProducerSaslPassword: saslPassword,
		ConsumerBrokers:      brokersCSV,
		ConsumerSaslUsername: saslUsername,
		ConsumerSaslPassword: saslPassword,
		ConsumerPartition:    consumerPartition,
	}, fill)
}

// SetConsumerPartitions sets o.Partitions from ConsumerPartition or HOSTNAME suffix.
func SetConsumerPartitions(o *RawKafkaOpts) error {
	if o == nil || len(o.Consumer.Brokers) == 0 {
		return nil
	}
	var p int
	var err error
	if o.ConsumerPartition != nil {
		p = *o.ConsumerPartition
	} else {
		p, err = kafka.GetPartitionFromHostname()
		if err != nil {
			return err
		}
	}
	o.Partitions = []int{p}
	return nil
}

// SetConsumerPartitionsFromHostname is deprecated; use SetConsumerPartitions.
func SetConsumerPartitionsFromHostname(o *RawKafkaOpts) error {
	return SetConsumerPartitions(o)
}

// SetRedisConfig configures Redis for the queue.
func SetRedisConfig(connectionURL string, maxLength int64) {
	redisstream.SetConfig(&redisstream.RedisConfig{
		ConnectionURL: connectionURL,
	})
	redisstream.SetProducerConfig(&redisstream.ProducerConfig{
		MaxLength:  maxLength,
		TTLSeconds: 14400,
	})
}

type hybridProducer struct {
	redis       *redisstream.Producer
	kafka       *kafka.KafkaProducer
	kafkaTopics map[string]struct{}
}

func hybridKafkaTopicSet(o *RawKafkaOpts) map[string]struct{} {
	m := make(map[string]struct{})
	add := func(s string) {
		if s != "" {
			m[s] = struct{}{}
		}
	}
	if o == nil {
		return m
	}
	add(o.TopicRawWeb)
	add(o.TopicRawMobile)
	add(o.TopicTrigger)
	add(o.TopicMobileTrigger)
	add(o.TopicRawImages)
	add(o.TopicCanvasImages)
	add(o.TopicCanvasTrigger)
	return m
}

func (h *hybridProducer) useKafka(topic string) bool {
	if h.kafka == nil || topic == "" {
		return false
	}
	_, ok := h.kafkaTopics[topic]
	return ok
}

func (h *hybridProducer) Produce(topic string, key uint64, value []byte) error {
	if h.useKafka(topic) {
		return h.kafka.Produce(topic, key, value)
	}
	return h.redis.Produce(topic, key, value)
}

func (h *hybridProducer) ProduceToPartition(topic string, partition, key uint64, value []byte) error {
	if h.useKafka(topic) {
		return h.kafka.ProduceToPartition(topic, partition, key, value)
	}
	return h.redis.ProduceToPartition(topic, partition, key, value)
}

func (h *hybridProducer) Flush(timeout int) {
	if h.kafka != nil {
		h.kafka.Flush(timeout)
	}
	h.redis.Flush(timeout)
}

func (h *hybridProducer) Close(timeout int) {
	if h.kafka != nil {
		h.kafka.Close(timeout)
	}
	h.redis.Close(timeout)
}

// NewProducer returns a Redis producer, or hybrid (listed topics -> Kafka) when producer brokers are set.
func NewProducer(_ int, _ bool, rawKafka *RawKafkaOpts) types.Producer {
	rp := redisstream.NewProducer()
	if !rawKafka.ProducerEnabled() {
		return rp
	}
	p := rawKafka.Producer
	kp, err := kafka.NewKafkaProducer(logger.New(), p.Brokers, kafka.ProducerConfig{
		RequiredAcks: 1,
		ClientConfig: kafka.ClientConfig{
			SASLUsername: p.SASLUsername,
			SASLPassword: p.SASLPassword,
		},
	})
	if err != nil {
		logger.New().Error(context.Background(), "kafka producer init failed (Redis fallback for non-Kafka topics): %v", err)
	}
	return &hybridProducer{
		redis:       rp,
		kafka:       kp,
		kafkaTopics: hybridKafkaTopicSet(rawKafka),
	}
}

// NewConsumer is the generic multi-topic Redis consumer (other services).
func NewConsumer(log logger.Logger, group string, topics []string, iterator messages.MessageIterator, _ bool, _ int, _ types.RebalanceHandler, _ time.Duration) (types.Consumer, error) {
	return redisstream.NewConsumer(group, topics, iterator)
}

// NewRawTopicConsumer is a single-topic consumer: Kafka when consumer brokers are set, else Redis.
func NewRawTopicConsumer(
	log logger.Logger,
	group string,
	topic string,
	iterator messages.MessageIterator,
	rebalanceHandler types.RebalanceHandler,
	rawKafka *RawKafkaOpts,
) (types.Consumer, error) {
	if rawKafka != nil && rawKafka.ConsumerEnabled() {
		if len(rawKafka.Partitions) == 0 {
			return nil, fmt.Errorf("kafka: partitions missing on RawKafkaOpts (call SetConsumerPartitions before NewRawTopicConsumer)")
		}
		c := rawKafka.Consumer
		return kafka.NewKafkaConsumer(
			log,
			c.Brokers,
			group,
			topic,
			rawKafka.Partitions,
			iterator,
			kafka.ClientConfig{
				SASLUsername: c.SASLUsername,
				SASLPassword: c.SASLPassword,
			},
			rebalanceHandler,
		)
	}
	return redisstream.NewConsumer(group, []string{topic}, iterator)
}
