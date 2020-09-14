package edp

import (
	"github.com/kyma-project/control-plane/components/metris/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	sentEvent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.Namespace,
			Subsystem: "edp",
			Name:      "request_total",
			Help:      "Total number of HTTP request made to EDP.",
		},
		[]string{"status"},
	)

	sentEventDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metrics.Namespace,
			Subsystem: "edp",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP request to EDP in seconds.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	)
)
