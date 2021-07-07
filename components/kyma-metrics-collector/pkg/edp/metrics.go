package edp

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	Namespace = "kmc"
	Subsystem = "edp"
)

var (
	totalRequest = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_total",
			Help:      "Total number of requests to EDP.",
		},
		[]string{"status"},
	)

	sentRequestDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP request to EDP in seconds.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	)
)
