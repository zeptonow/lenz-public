package middleware

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type ingestValidatorImpl struct{}

// NewIngestValidator adds lightweight request-shape validation for publicly exposed ingest routes.
// It does not enforce file-extension allowlists (those must be derived from real client behavior first).
func NewIngestValidator() *ingestValidatorImpl {
	return &ingestValidatorImpl{}
}

func (m *ingestValidatorImpl) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathTemplate, err := mux.CurrentRoute(r).GetPathTemplate()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if !isIngestV1Route(pathTemplate) {
			next.ServeHTTP(w, r)
			return
		}

		ce := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding")))
		if ce != "" && ce != "identity" && ce != "gzip" {
			http.Error(w, "unsupported content-encoding", http.StatusUnsupportedMediaType)
			return
		}

		ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))

		switch {
		case isStartRoute(pathTemplate) && r.Method == http.MethodPost:
			// Some clients may omit Content-Type; reject only obviously wrong types.
			if ct != "" && !strings.HasPrefix(ct, "application/json") {
				http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
				return
			}
		case isImagesRoute(pathTemplate) && r.Method == http.MethodPost:
			if !strings.HasPrefix(ct, "multipart/form-data") {
				http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
				return
			}
		case isIngestBinaryRoute(pathTemplate) && r.Method == http.MethodPost:
			// Explicitly reject multipart submissions on the binary ingest endpoints.
			if strings.HasPrefix(ct, "multipart/form-data") {
				http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
				return
			}
		}

		if batch := r.URL.Query().Get("batch"); batch != "" && len(batch) > 256 {
			http.Error(w, "invalid query param", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isIngestV1Route(pathTemplate string) bool {
	return strings.HasPrefix(pathTemplate, "/v1/web/") ||
		strings.HasPrefix(pathTemplate, "/v1/mobile/") ||
		strings.HasPrefix(pathTemplate, "/v1/sdk/")
}

func isStartRoute(pathTemplate string) bool {
	return strings.HasSuffix(pathTemplate, "/v1/web/start") ||
		strings.HasSuffix(pathTemplate, "/v1/mobile/start") ||
		strings.HasSuffix(pathTemplate, "/v1/sdk/start")
}

func isImagesRoute(pathTemplate string) bool {
	return strings.HasSuffix(pathTemplate, "/v1/web/images") ||
		strings.HasSuffix(pathTemplate, "/v1/mobile/images")
}

func isIngestBinaryRoute(pathTemplate string) bool {
	return strings.HasSuffix(pathTemplate, "/v1/web/i") ||
		strings.HasSuffix(pathTemplate, "/v1/mobile/i") ||
		strings.HasSuffix(pathTemplate, "/v1/mobile/late") ||
		strings.HasSuffix(pathTemplate, "/v1/sdk/i")
}
