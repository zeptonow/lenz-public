package metrics

import (
	"context"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"openreplay/backend/pkg/logger"
)

type MetricServer struct {
	registry *prometheus.Registry
}

var (
	metricsOnce    sync.Once
	globalRegistry *prometheus.Registry
	metricsStarted bool
)

func New(log logger.Logger, cs []prometheus.Collector) {
	metricsOnce.Do(func() {
		globalRegistry = prometheus.NewRegistry()
		// Add go runtime metrics and process collectors.
		globalRegistry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
		// Expose /metrics HTTP endpoint using the created custom registry.
		http.Handle(
			"/metrics", promhttp.HandlerFor(
				globalRegistry,
				promhttp.HandlerOpts{
					EnableOpenMetrics: true,
				}),
		)
		go func() {
			log.Error(context.Background(), "%v", http.ListenAndServe(":8888", nil))
		}()
		metricsStarted = true
	})

	// Register service-specific metrics to the global registry
	if metricsStarted && globalRegistry != nil {
		for _, c := range cs {
			if err := globalRegistry.Register(c); err != nil {
				// Ignore duplicate registration errors
				log.Info(context.Background(), "Metric already registered (ignored): %v", err)
			}
		}
	}
}
