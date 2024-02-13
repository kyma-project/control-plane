package commons

import (
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	SuccessListingSVCLabel   = "success_listing_svc"
	SuccessListingPVCLabel   = "success_listing_pvc"
	SuccessListingNodesLabel = "success_listing_nodes"
	SuccessStatusLabel       = "success"
	TotalQueriesLabel        = "query_total"
	ListingNodesAction       = "listing_nodes"
	ListingPVCAction         = "listing_pvc"
	ListingSVCAction         = "listing_svc"
)

var (
	TotalQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kmc",
			Subsystem: "skr",
			Name:      TotalQueriesLabel,
			Help:      "Total number of queries to SKR to get the metrics of the cluster.",
		},
		[]string{"action", "success", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)

func RecordSKRQuery(success bool, action string, shootInfo kmccache.Record) {
	TotalQueries.WithLabelValues(
		strconv.FormatBool(success),
		action,
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Inc()
}
