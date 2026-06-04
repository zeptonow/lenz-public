package queue

import "testing"

func TestNewRawKafkaOptsSplitProducerConsumer(t *testing.T) {
	o, err := NewRawKafkaOpts(KafkaSettings{
		ProducerBrokers:      "prod:9092",
		ProducerSaslUsername: "pu",
		ProducerSaslPassword: "pp",
		ConsumerBrokers:      "cons:9092",
		ConsumerSaslUsername: "cu",
		ConsumerSaslPassword: "cp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !o.ProducerEnabled() || !o.ConsumerEnabled() {
		t.Fatal("expected both enabled")
	}
	if o.Producer.Brokers[0] != "prod:9092" || o.Consumer.Brokers[0] != "cons:9092" {
		t.Fatalf("brokers: prod=%v cons=%v", o.Producer.Brokers, o.Consumer.Brokers)
	}
	if o.Producer.SASLUsername != "pu" || o.Consumer.SASLUsername != "cu" {
		t.Fatal("SASL username mismatch")
	}
}
