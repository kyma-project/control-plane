package metrics

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// OperationsStatsGetter provides metrics, which shows how many operations were done for the following plans:

// - compass_keb_operations_{plan_name}_provisioning_failed_total
// - compass_keb_operations_{plan_name}_provisioning_in_progress_total
// - compass_keb_operations_{plan_name}_provisioning_succeeded_total
// - compass_keb_operations_{plan_name}_deprovisioning_failed_total
// - compass_keb_operations_{plan_name}_deprovisioning_in_progress_total
// - compass_keb_operations_{plan_name}_deprovisioning_succeeded_total

var (
	supportedPlansIDs = []string{
		broker.AzurePlanID,
		broker.AzureLitePlanID,
		broker.AWSPlanID,
		broker.GCPPlanID,
		broker.OpenStackPlanID,
		broker.TrialPlanID,
		broker.FreemiumPlanID,
	}
)

type OperationsStatsGetter interface {
	GetOperationStatsByPlan() (map[string]internal.OperationStats, error)
}

type OperationStat struct {
	failedProvisioning   *prometheus.Desc
	failedDeprovisioning *prometheus.Desc

	succeededProvisioning   *prometheus.Desc
	succeededDeprovisioning *prometheus.Desc

	inProgressProvisioning   *prometheus.Desc
	inProgressDeprovisioning *prometheus.Desc
}

func (c *OperationStat) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.inProgressProvisioning
	ch <- c.succeededProvisioning
	ch <- c.failedProvisioning

	ch <- c.inProgressDeprovisioning
	ch <- c.succeededDeprovisioning
	ch <- c.failedDeprovisioning
}

type OperationsCollector struct {
	statsGetter OperationsStatsGetter

	operationStats map[string]OperationStat
}

func NewOperationsCollector(statsGetter OperationsStatsGetter) *OperationsCollector {
	opStats := make(map[string]OperationStat, len(supportedPlansIDs))

	for _, p := range supportedPlansIDs {
		opStats[p] = OperationStat{
			inProgressProvisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeProvision, domain.InProgress),
				"The number of provisioning operations in progress",
				[]string{"plan_id"},
				nil),
			succeededProvisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeProvision, domain.Succeeded),
				"The number of succeeded provisioning operations",
				[]string{"plan_id"},
				nil),
			failedProvisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeProvision, domain.Failed),
				"The number of failed provisioning operations",
				[]string{"plan_id"},
				nil),
			inProgressDeprovisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeDeprovision, domain.InProgress),
				"The number of deprovisioning operations in progress",
				[]string{"plan_id"},
				nil),
			succeededDeprovisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeDeprovision, domain.Succeeded),
				"The number of succeeded deprovisioning operations",
				[]string{"plan_id"},
				nil),
			failedDeprovisioning: prometheus.NewDesc(
				fqName(internal.OperationTypeDeprovision, domain.Failed),
				"The number of failed deprovisioning operations",
				[]string{"plan_id"},
				nil),
		}
	}

	return &OperationsCollector{
		statsGetter:    statsGetter,
		operationStats: opStats,
	}
}

func fqName(operationType internal.OperationType, state domain.LastOperationState) string {
	var opType string
	switch operationType {
	case internal.OperationTypeProvision:
		opType = "provisioning"
	case internal.OperationTypeDeprovision:
		opType = "deprovisioning"
	}

	var st string
	switch state {
	case domain.Failed:
		st = "failed"
	case domain.Succeeded:
		st = "succeeded"
	case domain.InProgress:
		st = "in_progress"
	}
	name := fmt.Sprintf("operations_%s_%s_total", opType, st)
	return prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, name)
}

func (c *OperationsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, op := range c.operationStats {
		op.Describe(ch)
	}
}

// Collect implements the prometheus.Collector interface.
func (c *OperationsCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.statsGetter.GetOperationStatsByPlan()
	if err != nil {
		return
	}

	for planID, ops := range c.operationStats {
		collect(ch,
			ops.inProgressProvisioning,
			stats[planID].Provisioning[domain.InProgress],
			planID,
		)
		collect(ch,
			ops.succeededProvisioning,
			stats[planID].Provisioning[domain.Succeeded],
			planID,
		)
		collect(ch,
			ops.failedProvisioning,
			stats[planID].Provisioning[domain.Failed],
			planID,
		)
		collect(ch,
			ops.inProgressDeprovisioning,
			stats[planID].Deprovisioning[domain.InProgress],
			planID,
		)
		collect(ch,
			ops.succeededDeprovisioning,
			stats[planID].Deprovisioning[domain.Succeeded],
			planID,
		)
		collect(ch,
			ops.failedDeprovisioning,
			stats[planID].Deprovisioning[domain.Failed],
			planID,
		)
	}

}

func collect(ch chan<- prometheus.Metric, desc *prometheus.Desc, value int, labelValues ...string) {
	m, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		float64(value),
		labelValues...)

	if err != nil {
		logrus.Errorf("unable to register metric %s", err.Error())
		return
	}
	ch <- m
}
