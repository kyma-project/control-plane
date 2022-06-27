package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	prometheusNamespace = "compass"
	prometheusSubsystem = "keb"

	resultFailed        float64 = 0
	resultSucceeded     float64 = 1
	resultInProgress    float64 = 2
	resultPending       float64 = 3
	resultCanceling     float64 = 4
	resultCanceled      float64 = 5
	resultRetrying      float64 = 6
	resultUnimplemented float64 = 7
)

type LastOperationState = domain.LastOperationState

const (
	Pending   LastOperationState = "pending"
	Canceling LastOperationState = "canceling"
	Canceled  LastOperationState = "canceled"
	Retrying  LastOperationState = "retrying"
)

// OperationResultCollector provides the following metrics:
// - compass_keb_provisioning_result{"operation_id", "instance_id", "global_account_id", "plan_id"}
// - compass_keb_deprovisioning_result{"operation_id", "instance_id", "global_account_id", "plan_id"}
// - compass_keb_upgrade_result{"operation_id", "instance_id", "global_account_id", "plan_id"}
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
	upgradeKymaResultGauge    *prometheus.GaugeVec
	upgradeClusterResultGauge *prometheus.GaugeVec
}

func NewOperationResultCollector() *OperationResultCollector {
	return &OperationResultCollector{
		provisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "provisioning_result",
			Help:      "Result of the provisioning",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "error_category", "error_reason"}),
		deprovisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "deprovisioning_result",
			Help:      "Result of the deprovisioning",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id", "error_category", "error_reason"}),
		upgradeKymaResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "upgrade_kyma_result",
			Help:      "Result of the kyma upgrade",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id"}),
		upgradeClusterResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "upgrade_cluster_result",
			Help:      "Result of the cluster upgrade",
		}, []string{"operation_id", "instance_id", "global_account_id", "plan_id"}),
	}
}

func (c *OperationResultCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningResultGauge.Describe(ch)
	c.deprovisioningResultGauge.Describe(ch)
	c.upgradeKymaResultGauge.Describe(ch)
	c.upgradeClusterResultGauge.Describe(ch)
}

func (c *OperationResultCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningResultGauge.Collect(ch)
	c.deprovisioningResultGauge.Collect(ch)
	c.upgradeKymaResultGauge.Collect(ch)
	c.upgradeClusterResultGauge.Collect(ch)
}

func (c *OperationResultCollector) OnUpgradeKymaStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.UpgradeKymaStepProcessed)
	if !ok {
		return fmt.Errorf("expected UpgradeKymaStepProcessed but got %+v", ev)
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
	case Retrying:
		resultValue = resultRetrying
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.upgradeKymaResultGauge.
		WithLabelValues(op.Operation.ID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).
		Set(resultValue)

	return nil
}

func (c *OperationResultCollector) OnUpgradeClusterStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.UpgradeClusterStepProcessed)
	if !ok {
		return fmt.Errorf("expected UpgradeClusterStepProcessed but got %+v", ev)
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
	case Retrying:
		resultValue = resultRetrying
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.upgradeClusterResultGauge.
		WithLabelValues(op.Operation.ID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).
		Set(resultValue)

	return nil
}

func (c *OperationResultCollector) OnProvisioningSucceeded(ctx context.Context, ev interface{}) error {
	provisioningSucceeded, ok := ev.(process.ProvisioningSucceeded)
	if !ok {
		return fmt.Errorf("expected ProvisioningSucceeded but got %+v", ev)
	}
	op := provisioningSucceeded.Operation
	pp := op.ProvisioningParameters
	c.provisioningResultGauge.WithLabelValues(
		op.ID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID, "", "").
		Set(resultSucceeded)

	return nil
}

func (c *OperationResultCollector) OnProvisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.ProvisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected ProvisioningStepProcessed but got %+v", ev)
	}

	var resultValue float64
	switch stepProcessed.Operation.State {
	case domain.InProgress, Pending, Retrying:
		resultValue = resultInProgress
	case domain.Succeeded:
		resultValue = resultSucceeded
	case domain.Failed, Canceling, Canceled:
		resultValue = resultFailed
	default:
		resultValue = resultFailed
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	err := op.LastError
	c.provisioningResultGauge.
		WithLabelValues(
			op.ID,
			op.InstanceID,
			pp.ErsContext.GlobalAccountID,
			pp.PlanID,
			string(err.Component()),
			string(err.Reason())).Set(resultValue)

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
	case Pending:
		resultValue = resultPending
	default:
		resultValue = resultUnimplemented
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	err := op.LastError
	c.deprovisioningResultGauge.
		WithLabelValues(
			op.ID,
			op.InstanceID,
			pp.ErsContext.GlobalAccountID,
			pp.PlanID,
			string(err.Component()),
			string(err.Reason())).Set(resultValue)
	return nil
}
