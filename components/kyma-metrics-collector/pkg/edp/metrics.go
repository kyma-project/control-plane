package edp

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

const (
	namespace = "kmc"
	subSystem = "edp"
	// responseCodeLabel name of the status code labels used by multiple metrics.
	responseCodeLabel = "status"
	// destSvcLabel name of the destination service label used by multiple metrics.
	requestURLLabel = "request_url"
	// metrics names.
	latencyMetricName = "request_duration_seconds"
)

var (
	latencyMetric = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subSystem,
			Name:      latencyMetricName,
			Help:      "Duration of HTTP request to EDP in seconds.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{responseCodeLabel, requestURLLabel},
	)
)

func recordEDPLatency(duration time.Duration, statusCode int, destSvc string) {
	latencyMetric.WithLabelValues(fmt.Sprint(statusCode), destSvc).Observe(duration.Seconds())
}
