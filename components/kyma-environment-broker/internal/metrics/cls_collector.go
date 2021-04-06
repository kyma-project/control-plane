package metrics

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// CLSInstancesStatsGetter provides number of all CLS instances in a region
type CLSInstancesStatsGetter interface {
	GetCLSInstanceStatsByRegion(region string) (int, error)
}

type ClsCollector struct {
	statsGetter   CLSInstancesStatsGetter
	clsDesc       *prometheus.Desc
	clsRegionDesc *prometheus.Desc
}

func NewClsCollector(statsGetter CLSInstancesStatsGetter) *ClsCollector {
	return &ClsCollector{
		statsGetter: statsGetter,

		clsDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "cls_instances_total"),
			"The total number of cls instances",
			[]string{},
			nil),

		clsRegionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(prometheusNamespace, prometheusSubsystem, "cls_instances_total_by_region"),
			"The total number of cls instances in a given region",
			[]string{"region"},
			nil),
	}
}

func (c *ClsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.clsDesc
	ch <- c.clsRegionDesc

}

// Collect implements the prometheus.Collector interface.
func (c *ClsCollector) Collect(ch chan<- prometheus.Metric) {
	totalCLSInstances := 0
	regions := cls.GetClsRegions()
	for _, region := range regions {
		clsIntancesCountByRegion, err := c.statsGetter.GetCLSInstanceStatsByRegion(region)
		totalCLSInstances = totalCLSInstances + clsIntancesCountByRegion
		if err != nil {
			logrus.Error(err)
			return
		}
		collect(ch, c.clsRegionDesc, clsIntancesCountByRegion)
	}
	collect(ch, c.clsDesc, totalCLSInstances)
}
