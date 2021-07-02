package commons

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	SuccessGettingSecretLabel = "success_getting_secret"
	SuccessGettingShootLabel  = "success_getting_shoot"
	FailedGettingShootLabel   = "failed_getting_shoot"
	FailedGettingSecretLabel  = "failed_getting_secret"
	SuccessStatusLabel        = "success"
	FailureStatusLabel        = "failure"
)

var (
	TotalCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kmc",
			Subsystem: "gardener",
			Name:      "calls_total",
			Help:      "Total number of calls to Gardener to get the config of the cluster.",
		},
		[]string{"status", "shoot", "reason"},
	)
)
