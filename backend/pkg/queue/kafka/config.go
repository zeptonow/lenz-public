package kafka

import (
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// ClientConfig holds optional Kafka client security settings (e.g. Confluent Cloud SASL_SSL).
type ClientConfig struct {
	SASLUsername string
	SASLPassword string
}

// JoinBootstrapServers returns a comma-separated bootstrap.servers value.
func JoinBootstrapServers(brokers []string) string {
	return strings.Join(brokers, ",")
}

// NormalizeBootstrapCSV parses broker CSV (strip URI schemes) and returns bootstrap.servers.
func NormalizeBootstrapCSV(brokersCSV string) string {
	return JoinBootstrapServers(ParseBrokerCSV(brokersCSV))
}

// ParseBrokerCSV splits comma-separated broker addresses (host:port only).
// Strips optional URI schemes copied from Confluent Console (e.g. SASL_SSL://host:9092).
func ParseBrokerCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if i := strings.Index(t, "://"); i >= 0 {
			t = t[i+3:]
		}
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// BuildConfigMap creates a librdkafka ConfigMap matching confluent_proto_producer.go in
// rider-management-service: bootstrap.servers plus optional SASL_SSL / PLAIN credentials.
func BuildConfigMap(bootstrapServers string, clientCfg ClientConfig, extra map[string]any) (*kafka.ConfigMap, error) {
	bootstrapServers = strings.TrimSpace(bootstrapServers)
	if bootstrapServers == "" {
		return nil, fmt.Errorf("kafka: bootstrap servers is empty")
	}

	cm := &kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	}
	for k, v := range extra {
		if err := cm.SetKey(k, v); err != nil {
			return nil, fmt.Errorf("kafka config %q: %w", k, err)
		}
	}
	if err := applySASLConfig(cm, clientCfg); err != nil {
		return nil, err
	}
	return cm, nil
}

func applySASLConfig(cm *kafka.ConfigMap, cfg ClientConfig) error {
	if cfg.SASLUsername == "" && cfg.SASLPassword == "" {
		return nil
	}
	if cfg.SASLUsername == "" || cfg.SASLPassword == "" {
		return fmt.Errorf("kafka: both SASL username and password are required when either is set")
	}
	if err := cm.SetKey("security.protocol", "SASL_SSL"); err != nil {
		return err
	}
	if err := cm.SetKey("sasl.mechanisms", "PLAIN"); err != nil {
		return err
	}
	if err := cm.SetKey("sasl.username", cfg.SASLUsername); err != nil {
		return err
	}
	if err := cm.SetKey("sasl.password", cfg.SASLPassword); err != nil {
		return err
	}
	return nil
}

// ApplySASLConfig enables SASL_SSL when both username and password are set.
// Deprecated: use BuildConfigMap.
func ApplySASLConfig(cm *kafka.ConfigMap, cfg ClientConfig) error {
	return applySASLConfig(cm, cfg)
}
