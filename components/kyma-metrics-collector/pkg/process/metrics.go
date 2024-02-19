package process

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "kmc"
	subsystem = "process"
	// requestURLLabel name of the request URL label used by multiple metrics.
	requestURLLabel = "request_url"
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
	kebTotalClusters = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "clusters_total",
			Help:      "Number of all clusters got from KEB including trackable and not trackable.",
		},
		[]string{"trackable", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)

func recordKEBTotalClusters(trackable bool, shootName, instanceID, runtimeID, subAccountID, globalAccountID string) {
	// the order if the values should be same as defined in the metric declaration.
	kebTotalClusters.WithLabelValues(strconv.FormatBool(trackable), shootName, instanceID, runtimeID, subAccountID, globalAccountID).Inc()
}
