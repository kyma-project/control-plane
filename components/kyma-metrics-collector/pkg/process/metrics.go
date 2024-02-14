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
	kebAllClustersCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kmc",
			Subsystem: "keb",
			Name:      "all_clusters_count",
			Help:      "Number of all clusters got from KEB.",
		},
		[]string{"state", "provisioning", "deprovisioning", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)
