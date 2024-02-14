package keb

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

const (
	Namespace = "kmc"
	Subsystem = "keb"
	// responseCodeLabel name of the status code labels used by multiple metrics.
	responseCodeLabel = "status"
	// requestURLLabel name of the request URL label used by multiple metrics.
	requestURLLabel = "request_url"
	// metrics names.
	latencyMetricName = "request_duration_seconds"
)

var (
	latencyMetric = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: Subsystem,
			Name:      latencyMetricName,
			Help:      "Duration of HTTP request to KEB in seconds.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{responseCodeLabel, requestURLLabel},
	)
)

func recordKEBLatency(duration time.Duration, statusCode int, destSvc string) {
	// the order if the values should be same as defined in the metric declaration.
	latencyMetric.WithLabelValues(fmt.Sprint(statusCode), destSvc).Observe(duration.Seconds())
}
