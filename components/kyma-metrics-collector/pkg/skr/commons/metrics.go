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
	shootNameLabel         = "shoot_name"
	instanceIdLabel        = "instance_id"
	runtimeIdLabel         = "runtime_id"
	subAccountLabel        = "sub_account_id"
	globalAccountLabel     = "global_account_id"
	successLabel           = "success"
	actionLabel            = "action"
)

var (
	TotalQueriesMetric = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      TotalQueriesMetricName,
			Help:      "Total number of queries to SKR to get the metrics of the cluster.",
		},
		[]string{actionLabel, successLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)
)

// DeleteMetrics deletes all the metrics for the provided shoot.
// Returns true if some metrics are deleted, returns false if no metrics are deleted for that subAccount.
func DeleteMetrics(shootInfo kmccache.Record) bool {
	matchLabels := prometheus.Labels{
		shootNameLabel:     shootInfo.ShootName,
		instanceIdLabel:    shootInfo.InstanceID,
		runtimeIdLabel:     shootInfo.RuntimeID,
		subAccountLabel:    shootInfo.SubAccountID,
		globalAccountLabel: shootInfo.GlobalAccountID,
	}
	return TotalQueriesMetric.DeletePartialMatch(matchLabels) > 0
}

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
