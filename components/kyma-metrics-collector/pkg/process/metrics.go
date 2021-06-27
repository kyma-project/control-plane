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
)
