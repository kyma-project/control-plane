package storage

import (
	"github.com/kyma-project/control-plane/components/metris/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	storageItemCountMetricVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Subsystem: "storage",
			Name:      "item_count",
			Help:      "Actual number of item currently in storage.",
		},
		[]string{"name"},
	)
)
