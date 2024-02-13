package edp

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

const (
	Namespace = "kmc"
	Subsystem = "edp"
)

var (
	sentRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP request to EDP in seconds.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"status", "destination_service"},
	)
)

func recordEDPLatency(duration time.Duration, statusCode int, destSvc string) {
	sentRequestDuration.WithLabelValues(fmt.Sprint(statusCode), destSvc).Observe(duration.Seconds())
}
