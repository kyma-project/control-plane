package process

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	clustersScraped = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kmc",
			Subsystem: "keb",
			Name:      "number_clusters_scraped",
			Help:      "Number of clusters scraped.",
		},
		[]string{"requestURI"},
	)
	numberClusters = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kmc",
			Subsystem: "process",
			Name:      "number_clusters",
			Help:      "Number of all clusters.",
		},
		[]string{"status", "shoot", "instanceid", "runtimeid", "subaccountid", "globalaccountid"},
	)
)
