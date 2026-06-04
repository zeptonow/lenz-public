package queue

import (
	"testing"

	"openreplay/backend/pkg/queue/kafka"
)

func TestParseBrokerCSVDelegatesToKafka(t *testing.T) {
	got := ParseBrokerCSV("SASL_SSL://lkc-abc.ap-south-1.aws.glb.confluent.cloud:9092")
	want := kafka.ParseBrokerCSV("SASL_SSL://lkc-abc.ap-south-1.aws.glb.confluent.cloud:9092")
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("ParseBrokerCSV = %v, want %v", got, want)
	}
}
