package azure

import (
	"github.com/kyma-project/control-plane/components/metris/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	apiRequestDurationHist = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metrics.Namespace,
			Subsystem: "azure",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP request to Azure in seconds.",
			// Duration buckets, in seconds
			// 500ms, 600ms, 700ms, 800ms, 900ms, 1s, 1.25s, 1.5s, 1.75s, 2s
			Buckets: []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1, 1.25, 1.5, 1.75, 2.0},
		},
		[]string{"provider", "resource"},
	)

	apiRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.Namespace,
			Subsystem: "azure",
			Name:      "request_total",
			Help:      "Total number of HTTP request made to Azure.",
		},
		[]string{"provider", "resource", "status"},
	)
)

// collectRequestMetrics collect Azure REST HTTP request metrics.
func collectRequestMetrics(provider, resource string) func() {
	metricTimer := prometheus.NewTimer(apiRequestDurationHist.WithLabelValues(provider, resource))

	return func() {
		_ = metricTimer.ObserveDuration()
	}
}
