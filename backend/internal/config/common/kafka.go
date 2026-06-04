package common

import "strings"

// Kafka is shared queue broker settings (Confluent Cloud, local broker, etc.).
// Embed with mapstructure:",squash" so keys stay at the config file root.
//
// Producer- and consumer-specific keys override the legacy shared kafkaBrokers /
// kafkaSaslUsername / kafkaSaslPassword when set.
type Kafka struct {
	// Legacy shared (fallback when producer/consumer-specific values are empty)
	KafkaBrokers      string `mapstructure:"kafkaBrokers"`
	KafkaSaslUsername string `mapstructure:"kafkaSaslUsername"`
	KafkaSaslPassword string `mapstructure:"kafkaSaslPassword"`

	KafkaProducerBrokers      string `mapstructure:"kafkaProducerBrokers"`
	KafkaProducerSaslUsername string `mapstructure:"kafkaProducerSaslUsername"`
	KafkaProducerSaslPassword string `mapstructure:"kafkaProducerSaslPassword"`

	KafkaConsumerBrokers      string `mapstructure:"kafkaConsumerBrokers"`
	KafkaConsumerSaslUsername string `mapstructure:"kafkaConsumerSaslUsername"`
	KafkaConsumerSaslPassword string `mapstructure:"kafkaConsumerSaslPassword"`
	KafkaConsumerPartition    *int   `mapstructure:"kafkaConsumerPartition"` // when set, overrides HOSTNAME suffix
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// KafkaEnabled is true when producer and/or consumer bootstrap servers are configured.
func (k *Kafka) KafkaEnabled() bool {
	return k.ProducerBrokers() != "" || k.ConsumerBrokers() != ""
}

func (k *Kafka) ProducerBrokers() string {
	return firstNonEmpty(k.KafkaProducerBrokers, k.KafkaBrokers)
}

func (k *Kafka) ProducerSaslUsername() string {
	return firstNonEmpty(k.KafkaProducerSaslUsername, k.KafkaSaslUsername)
}

func (k *Kafka) ProducerSaslPassword() string {
	if strings.TrimSpace(k.KafkaProducerSaslPassword) != "" {
		return k.KafkaProducerSaslPassword
	}
	return k.KafkaSaslPassword
}

func (k *Kafka) ConsumerBrokers() string {
	return firstNonEmpty(k.KafkaConsumerBrokers, k.KafkaBrokers)
}

func (k *Kafka) ConsumerSaslUsername() string {
	return firstNonEmpty(k.KafkaConsumerSaslUsername, k.KafkaSaslUsername)
}

func (k *Kafka) ConsumerSaslPassword() string {
	if strings.TrimSpace(k.KafkaConsumerSaslPassword) != "" {
		return k.KafkaConsumerSaslPassword
	}
	return k.KafkaSaslPassword
}
