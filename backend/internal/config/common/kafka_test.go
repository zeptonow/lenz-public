package common

import "testing"

func TestKafkaProducerConsumerFallback(t *testing.T) {
	k := Kafka{
		KafkaBrokers:              "shared:9092",
		KafkaSaslUsername:         "user",
		KafkaSaslPassword:         "pass",
		KafkaProducerBrokers:      "prod:9092",
		KafkaConsumerSaslUsername: "cons-user",
	}
	if k.ProducerBrokers() != "prod:9092" {
		t.Fatalf("ProducerBrokers = %q", k.ProducerBrokers())
	}
	if k.ConsumerBrokers() != "shared:9092" {
		t.Fatalf("ConsumerBrokers = %q", k.ConsumerBrokers())
	}
	if k.ConsumerSaslUsername() != "cons-user" {
		t.Fatalf("ConsumerSaslUsername = %q", k.ConsumerSaslUsername())
	}
	if k.ProducerSaslUsername() != "user" {
		t.Fatalf("ProducerSaslUsername = %q", k.ProducerSaslUsername())
	}
}

func TestKafkaEnabled(t *testing.T) {
	if (&Kafka{}).KafkaEnabled() {
		t.Fatal("expected disabled")
	}
	if !(&Kafka{KafkaProducerBrokers: "a:9092"}).KafkaEnabled() {
		t.Fatal("expected enabled via producer")
	}
}
