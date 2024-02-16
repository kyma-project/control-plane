package process

import (
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	namespace = "kmc"
	subsystem = "process"
)

var (
	subAccountProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sub_account_total",
			Help:      "Number of sub-accounts processed including successful and failed.",
		},
		[]string{"success", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
	subAccountProcessedTimeStamp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sub_account_processed_timestamp_seconds",
			Help:      "Unix timestamp (in seconds) of last successful processing of sub-account.",
		},
		[]string{"with_old_metric", "shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
	oldMetricsPublishedGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "old_metric_publish_gauge",
			Help:      "Number of consecutive re-sends of old metrics to edp per cluster. It Will be reset to 0 when new metric data is published.",
		},
		[]string{"shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)

// deleteMetrics deletes all the metrics for the provided shoot.
// Returns true if some metrics are deleted, returns false if no metrics are deleted for that subAccount.
func deleteMetrics(shootInfo kmccache.Record) bool {
	matchLabels := prometheus.Labels{
		"shoot_name":        shootInfo.ShootName,
		"instance_id":       shootInfo.InstanceID,
		"runtime_id":        shootInfo.RuntimeID,
		"sub_account_id":    shootInfo.SubAccountID,
		"global_account_id": shootInfo.GlobalAccountID,
	}

	count := 0 // total numbers of metrics deleted
	count += subAccountProcessed.DeletePartialMatch(matchLabels)
	count += subAccountProcessedTimeStamp.DeletePartialMatch(matchLabels)
	count += oldMetricsPublishedGauge.DeletePartialMatch(matchLabels)

	// delete metrics for SKR queries.
	return skrcommons.DeleteMetrics(shootInfo) && count > 0
}

func recordSubAccountProcessed(success bool, shootInfo kmccache.Record) {
	// the order if the values should be same as defined in the metric declaration.
	subAccountProcessed.WithLabelValues(
		strconv.FormatBool(success),
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Inc()
}

func recordSubAccountProcessedTimeStamp(withOldMetric bool, shootInfo kmccache.Record) {
	// the order if the values should be same as defined in the metric declaration.
	subAccountProcessedTimeStamp.WithLabelValues(
		strconv.FormatBool(withOldMetric),
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).SetToCurrentTime()
}

func recordOldMetricsPublishedGauge(shootInfo kmccache.Record) {
	// the order if the values should be same as defined in the metric declaration.
	oldMetricsPublishedGauge.WithLabelValues(
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Inc()
}

func resetOldMetricsPublishedGauge(shootInfo kmccache.Record) {
	// the order if the values should be same as defined in the metric declaration.
	oldMetricsPublishedGauge.WithLabelValues(
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Set(0.0)
}
