package metrics

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/metrics/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InProgressOperationsCollector_Collect(t *testing.T) {

	operationsCounts := model.OperationsCount{
		Count: map[model.OperationType]int{
			model.Provision:            10,
			model.ProvisionNoInstall:   6,
			model.Deprovision:          5,
			model.DeprovisionNoInstall: 3,
			model.Upgrade:              2,
		},
	}

	statsGetter := &mocks.OperationsStatsGetter{}
	statsGetter.On("InProgressOperationsCount").Return(operationsCounts, nil)

	collector := NewInProgressOperationsCollector(statsGetter)

	receiver := make(chan prometheus.Metric, 5)
	defer close(receiver)

	collector.Collect(receiver)

	provisionMetric := <-receiver
	assertGaugeValue(t, provisionMetric, float64(10))
	assert.Contains(t, provisionMetric.Desc().String(), "kcp_provisioner_in_progress_provision_operations_total")

	provisionNoInstallMetric := <-receiver
	assertGaugeValue(t, provisionNoInstallMetric, float64(6))
	assert.Contains(t, provisionNoInstallMetric.Desc().String(), "kcp_provisioner_in_progress_provision_no_install_operations_total")

	deprovisionMetric := <-receiver
	assertGaugeValue(t, deprovisionMetric, float64(5))
	assert.Contains(t, deprovisionMetric.Desc().String(), "kcp_provisioner_in_progress_deprovision_operations_total")

	deprovisionNoInstallMetric := <-receiver
	assertGaugeValue(t, deprovisionNoInstallMetric, float64(3))
	assert.Contains(t, deprovisionNoInstallMetric.Desc().String(), "kcp_provisioner_in_progress_deprovision_no_install_operations_total")

	upgradeMetric := <-receiver
	assertGaugeValue(t, upgradeMetric, float64(2))
	assert.Contains(t, upgradeMetric.Desc().String(), "kcp_provisioner_in_progress_upgrade_operations_total")
}

func Test_InProgressOperationsCollector_Describe(t *testing.T) {
	collector := NewInProgressOperationsCollector(nil)

	receiver := make(chan *prometheus.Desc, 5)
	defer close(receiver)

	collector.Describe(receiver)

	provisionDesc := <-receiver
	assert.Contains(t, provisionDesc.String(), "kcp_provisioner_in_progress_provision_operations_total")

	provisionNoInstallDesc := <-receiver
	assert.Contains(t, provisionNoInstallDesc.String(), "kcp_provisioner_in_progress_provision_no_install_operations_total")

	deprovisionDesc := <-receiver
	assert.Contains(t, deprovisionDesc.String(), "kcp_provisioner_in_progress_deprovision_operations_total")

	deprovisionNoInstallDesc := <-receiver
	assert.Contains(t, deprovisionNoInstallDesc.String(), "kcp_provisioner_in_progress_deprovision_no_install_operations_total")

	upgradeDesc := <-receiver
	assert.Contains(t, upgradeDesc.String(), "kcp_provisioner_in_progress_upgrade_operations_total")
}

func assertGaugeValue(t *testing.T, metric prometheus.Metric, expected float64) {
	metricDto := dto.Metric{}
	err := metric.Write(&metricDto)
	require.NoError(t, err)

	require.NotNil(t, metricDto.Gauge)
	require.NotNil(t, metricDto.Gauge.Value)
	assert.Equal(t, expected, *metricDto.Gauge.Value)
}
