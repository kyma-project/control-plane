package gardener

import (
	"github.com/kyma-project/control-plane/components/metris/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	clusterSyncErrorVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metrics.Namespace,
			Subsystem: "gardener",
			Name:      "error_total",
			Help:      "Total number of failed cluster syncs.",
		},
		[]string{"cause"},
	)
)
