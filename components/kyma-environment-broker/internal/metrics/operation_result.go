package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "compass"
	prometheusSubsystem = "keb"

	resultFailed     float64 = 0
	resultSucceeded  float64 = 1
	resultInProgress float64 = 2
	resultPending    float64 = 3
	resultCanceling  float64 = 4
	resultCanceled   float64 = 5
)

type LastOperationState = domain.LastOperationState

const (
	Pending   LastOperationState = "pending"
	Canceling LastOperationState = "canceling"
	Canceled  LastOperationState = "canceled"
)

// OperationResultCollector provides the following metrics:
// - compass_keb_provisioning_result{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}
// - compass_keb_deprovisioning_result{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}
// - compass_keb_upgrade_result{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}
// These gauges show the status of the operation.
// The value of the gauge could be:
// 0 - Failed
// 1 - Succeeded
// 2 - In progress
// 3 - Pending
// 4 - Canceling
// 5 - Canceled
type OperationResultCollector struct {
	provisioningResultGauge   *prometheus.GaugeVec
	deprovisioningResultGauge *prometheus.GaugeVec
	upgradeResultGauge        *prometheus.GaugeVec
}

func NewOperationResultCollector() *OperationResultCollector {
	return &OperationResultCollector{
		provisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "provisioning_result",
			Help:      "Result of the provisioning",
		}, []string{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}),
		deprovisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "deprovisioning_result",
			Help:      "Result of the deprovisioning",
		}, []string{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}),
		upgradeResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "upgrade_result",
			Help:      "Result of the upgrade",
		}, []string{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}),
	}
}

func (c *OperationResultCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningResultGauge.Describe(ch)
	c.deprovisioningResultGauge.Describe(ch)
	c.upgradeResultGauge.Describe(ch)
}

func (c *OperationResultCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningResultGauge.Collect(ch)
	c.deprovisioningResultGauge.Collect(ch)
	c.upgradeResultGauge.Collect(ch)
}

func (c *OperationResultCollector) OnUpgradeStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.UpgradeKymaStepProcessed)
	if !ok {
		return fmt.Errorf("expected UpgradeStepProcessed but got %+v", ev)
	}

	var resultValue float64
	switch stepProcessed.Operation.State {
	case domain.InProgress:
		resultValue = resultInProgress
	case domain.Succeeded:
		resultValue = resultSucceeded
	case domain.Failed:
		resultValue = resultFailed
	case Pending:
		resultValue = resultPending
	case Canceling:
		resultValue = resultCanceling
	case Canceled:
		resultValue = resultCanceled
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.upgradeResultGauge.
		WithLabelValues(op.ID, op.RuntimeID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).
		Set(resultValue)

	return nil
}

func (c *OperationResultCollector) OnProvisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.ProvisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected ProvisioningStepProcessed but got %+v", ev)
	}

	var resultValue float64
	switch stepProcessed.Operation.State {
	case domain.InProgress:
		resultValue = resultInProgress
	case domain.Succeeded:
		resultValue = resultSucceeded
	case domain.Failed:
		resultValue = resultFailed
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.provisioningResultGauge.
		WithLabelValues(op.ID, op.RuntimeID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).
		Set(resultValue)

	return nil
}

func (c *OperationResultCollector) OnDeprovisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.DeprovisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected DeprovisioningStepProcessed but got %+v", ev)
	}
	var resultValue float64
	switch stepProcessed.Operation.State {
	case domain.InProgress:
		resultValue = resultInProgress
	case domain.Succeeded:
		resultValue = resultSucceeded
	case domain.Failed:
		resultValue = resultFailed
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.deprovisioningResultGauge.
		WithLabelValues(op.ID, op.RuntimeID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).
		Set(resultValue)
	return nil
}
