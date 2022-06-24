package metrics

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// InstancesStatsGetter provides number of all instances failed, succeeded or orphaned
//   (instance exists but the cluster was removed manually from the gardener):
// - compass_keb_instances_total - total number of all instances
// - compass_keb_global_account_id_instances_total - total number of all instances per global account
// - compass_keb_ers_context_license_type_total - count of instances grouped by license types
type InstancesStatsGetter interface {
	GetInstanceStats() (internal.InstanceStats, error)
	GetERSContextStats() (internal.ERSContextStats, error)
}

type InstancesCollector struct {
	statsGetter InstancesStatsGetter

	instancesDesc        *prometheus.Desc
	instancesPerGAIDDesc *prometheus.Desc
	licenseTypeDesc      *prometheus.Desc
}

func NewInstancesCollector(statsGetter InstancesStatsGetter) *InstancesCollector {
	return &InstancesCollector{
		statsGetter: statsGetter,

		instancesDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "instances_total"),
			"The total number of instances",
			[]string{},
			nil),
		instancesPerGAIDDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "global_account_id_instances_total"),
			"The total number of instances by Global Account ID",
			[]string{"global_account_id"},
			nil),
		licenseTypeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "ers_context_license_type_total"),
			"count of instances grouped by license types",
			[]string{"license_type"},
			nil),
	}
}

func (c *InstancesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.instancesDesc
	ch <- c.instancesPerGAIDDesc
	ch <- c.licenseTypeDesc
}

// Collect implements the prometheus.Collector interface.
func (c *InstancesCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := c.statsGetter.GetInstanceStats()
	if err != nil {
		logrus.Error(err)
	} else {
		collect(ch, c.instancesDesc, stats.TotalNumberOfInstances)

		for globalAccountID, num := range stats.PerGlobalAccountID {
			collect(ch, c.instancesPerGAIDDesc, num, globalAccountID)
		}
	}

	stats2, err := c.statsGetter.GetERSContextStats()
	if err != nil {
		logrus.Error(err)
		return
	}
	for t, num := range stats2.LicenseType {
		collect(ch, c.licenseTypeDesc, num, t)
	}
}
