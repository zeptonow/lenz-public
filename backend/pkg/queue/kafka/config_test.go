package kafka

import "testing"

func TestBuildConfigMapSASL(t *testing.T) {
	cm, err := BuildConfigMap("host:9092", ClientConfig{SASLUsername: "key", SASLPassword: "secret"}, map[string]any{
		"acks": "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := (*cm)["security.protocol"]; v != "SASL_SSL" {
		t.Fatalf("security.protocol = %v", v)
	}
	if v, _ := (*cm)["sasl.username"]; v != "key" {
		t.Fatalf("sasl.username = %v", v)
	}
}

func TestBuildConfigMapPlaintextWithoutSASL(t *testing.T) {
	cm, err := BuildConfigMap("localhost:9092", ClientConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := (*cm)["security.protocol"]; ok {
		t.Fatal("expected no security.protocol without SASL")
	}
}

func TestParseBrokerCSVStripsScheme(t *testing.T) {
	got := ParseBrokerCSV("SASL_SSL://lkc-abc.ap-south-1.aws.glb.confluent.cloud:9092")
	if len(got) != 1 || got[0] != "lkc-abc.ap-south-1.aws.glb.confluent.cloud:9092" {
		t.Fatalf("ParseBrokerCSV = %v", got)
	}
	if NormalizeBootstrapCSV("SASL_SSL://a:9092,SASL_SSL://b:9092") != "a:9092,b:9092" {
		t.Fatal("NormalizeBootstrapCSV")
	}
}

func TestBuildConfigMapPartialSASLFails(t *testing.T) {
	_, err := BuildConfigMap("host:9092", ClientConfig{SASLUsername: "key"}, nil)
	if err == nil {
		t.Fatal("expected error for password-only partial SASL")
	}
}
