package commons

import (
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	namespace              = "kmc"
	subsystem              = "skr"
	TotalQueriesMetricName = "query_total"
	ListingNodesAction     = "listing_nodes"
	ListingPVCsAction      = "listing_pvc"
	ListingSVCsAction      = "listing_svc"
)

var (
	TotalQueriesMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      TotalQueriesMetricName,
			Help:      "Total number of queries to SKR to get the metrics of the cluster.",
		},
		[]string{"action", "success", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)

func RecordSKRQuery(success bool, action string, shootInfo kmccache.Record) {
	// the order if the values should be same as defined in the metric declaration.
	TotalQueriesMetric.WithLabelValues(
		action,
		strconv.FormatBool(success),
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Inc()
}
