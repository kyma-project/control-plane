package process

import (
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/commons"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	namespace          = "kmc"
	subsystem          = "process"
	shootNameLabel     = "shoot_name"
	instanceIdLabel    = "instance_id"
	runtimeIdLabel     = "runtime_id"
	subAccountLabel    = "sub_account_id"
	globalAccountLabel = "global_account_id"
	successLabel       = "success"
	withOldMetricLabel = "with_old_metric"
)

var (
	subAccountProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sub_account_total",
			Help:      "Number of sub-accounts processed including successful and failed.",
		},
		[]string{successLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)
	subAccountProcessedTimeStamp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sub_account_processed_timestamp_seconds",
			Help:      "Unix timestamp (in seconds) of last successful processing of sub-account.",
		},
		[]string{withOldMetricLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)
	oldMetricsPublishedGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "old_metric_published_gauge",
			Help:      "Number of consecutive re-sends of old metrics to edp per cluster. It Will reset to 0 when new metric data is published.",
		},
		[]string{shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
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
