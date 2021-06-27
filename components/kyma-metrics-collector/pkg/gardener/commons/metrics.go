package commons

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GardenerCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kmc",
			Subsystem: "gardener",
			Name:      "calls_total",
			Help:      "Total number of calls to Gardener to get the config of the cluster.",
		},
		[]string{"status", "shoot", "reason"},
	)
)
