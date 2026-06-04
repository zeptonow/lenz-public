package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/getsentry/sentry-go"
)

type extraLogger struct {
	hasSentry     bool
	hasElastic    bool
	dataDogAPIKey string
	elasticLogger *elasticsearch.Client
}

type ExtraLogger interface {
	Log(ctx context.Context, log string)
}

func NewExtraLogger() ExtraLogger {
	// Replay-only build: disable third-party log shipping by default.
	// To re-enable later, set LOG_SHIPPING_ENABLED=true.
	if os.Getenv("LOG_SHIPPING_ENABLED") != "true" {
		return &extraLogger{
			hasSentry:     false,
			hasElastic:    false,
			elasticLogger: nil,
			dataDogAPIKey: "",
		}
	}

	// Init sentry
	hasSentry := true
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN == "" {
		hasSentry = false
	} else {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDSN,
			TracesSampleRate: 1.0,
		})
		if err != nil {
			fmt.Printf("sentry.Init: %s", err)
			hasSentry = false
		}
	}

	// Init elasticsearch
	elasticHost := os.Getenv("ELASTIC_HOST")
	elasticAPIKey := os.Getenv("ELASTIC_API_KEY")

	hasElastic := true
	if elasticHost == "" {
		hasElastic = false
	}
	var es *elasticsearch.Client
	if hasElastic {
		var err error
		es, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses: []string{elasticHost},
			APIKey:    elasticAPIKey,
		})
		if err != nil {
			fmt.Printf("Error creating the ES client: %s", err)
			hasElastic = false
		}
	}

	dataDogAPIKey := os.Getenv("DATADOG_API_KEY")

	return &extraLogger{
		hasSentry:     hasSentry,
		hasElastic:    hasElastic,
		elasticLogger: es,
		dataDogAPIKey: dataDogAPIKey,
	}
}

// LogMessage defines the structure of your log message
type LogMessage struct {
	Timestamp time.Time `json:"@timestamp"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
}

func sendLog(es *elasticsearch.Client, logMessage LogMessage) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(logMessage); err != nil {
		fmt.Printf("Error encoding log message: %s", err)
		return
	}

	req := esapi.IndexRequest{
		Index:      "logs",
		DocumentID: "",
		Body:       &buf,
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		fmt.Printf("Error sending log to Elasticsearch: %s", err)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		fmt.Printf("Error response from Elasticsearch: %s", res.String())
	} else {
		fmt.Printf("Log successfully sent to Elasticsearch.")
	}
}

func (el *extraLogger) Log(ctx context.Context, msg string) {
	if sID, ok := ctx.Value("sessionID").(string); ok {
		msg = fmt.Sprintf("%s openReplaySession.id=%s", msg, sID)
	}
	if el.hasSentry {
		sentry.CaptureMessage(msg)
	}
	if el.hasElastic {
		esMsg := LogMessage{
			Timestamp: time.Now(),
			Message:   msg,
			Level:     "INFO",
		}
		sendLog(el.elasticLogger, esMsg)
	}
	if el.dataDogAPIKey == "" {
		return
	}
	url := "https://http-intake.logs.datadoghq.com/v1/input"

	type ddLog struct {
		Message  string `json:"message"`
		DDSource string `json:"ddsource"`
		Service  string `json:"service"`
		Hostname string `json:"hostname"`
		DDTags   string `json:"ddtags"`
	}

	payload := ddLog{
		Message:  msg,
		DDSource: "go",
		Service:  "myservice",
		Hostname: "myhost",
		DDTags:   "env:development",
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", el.dataDogAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send log to DataDog:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to send log to DataDog, status code:", resp.StatusCode)
	} else {
		fmt.Println("Log sent to DataDog successfully!")
	}

}
