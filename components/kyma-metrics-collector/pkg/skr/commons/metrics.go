package commons

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SkrCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kmc",
			Subsystem: "skr",
			Name:      "calls_total",
			Help:      "Total number of calls to SKR to get the metrics of the cluster.",
		},
		[]string{"status", "reason"},
	)
)
