package process

import (
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	namespace = "kmc"
	subsystem = "process"
	// requestURLLabel name of the request URL label used by multiple metrics.
	requestURLLabel = "request_url"
)

var (
	subAccountProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sub_account_total",
			Help:      "Number of sub-accounts processed.",
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
			Help:      "Number of consecutive re-sends of old metrics to edp per cluster. It Will be reset to 0 when new metric is published.",
		},
		[]string{"shoot_name", "instance_id", "runtime_id", "sub_account_id", "global_account_id"},
	)
)

func recordSubAccountProcessed(success bool, shootInfo kmccache.Record) {
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
	oldMetricsPublishedGauge.WithLabelValues(
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Inc()
}

func resetOldMetricsPublishedGauge(shootInfo kmccache.Record) {
	oldMetricsPublishedGauge.WithLabelValues(
		shootInfo.ShootName,
		shootInfo.InstanceID,
		shootInfo.RuntimeID,
		shootInfo.SubAccountID,
		shootInfo.GlobalAccountID,
	).Set(0.0)
}
