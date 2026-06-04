package httpclient

import (
	"net/http"
	"time"
)

const InterserviceAPIKeyHeader = "X-Api-Key"

type apiKeyTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.base == nil {
		t.base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	if t.apiKey != "" {
		clone.Header.Set(InterserviceAPIKeyHeader, t.apiKey)
	}
	return t.base.RoundTrip(clone)
}

func WithInterserviceAPIKey(base http.RoundTripper, apiKey string) http.RoundTripper {
	return &apiKeyTransport{apiKey: apiKey, base: base}
}

func NewInterserviceClient(apiKey string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: WithInterserviceAPIKey(http.DefaultTransport, apiKey),
	}
}
