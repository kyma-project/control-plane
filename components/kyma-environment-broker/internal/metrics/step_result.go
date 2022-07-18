package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// StepResultCollector provides the following metrics:
// - compass_keb_provisioning_step_result{"operation_id",  "instance_id", "step_name", "global_account_id", "plan_id"}
// - compass_keb_deprovisioning_step_result{"operation_id",  "instance_id", "step_name", "global_account_id", "plan_id"}
// These gauges show the status of the operation step.
// The value of the gauge could be:
// 0 - Failed
// 1 - Succeeded
// 2 - In progress
type StepResultCollector struct {
	provisioningResultGauge   *prometheus.GaugeVec
	deprovisioningResultGauge *prometheus.GaugeVec
}

func NewStepResultCollector() *StepResultCollector {
	return &StepResultCollector{
		provisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "provisioning_step_result",
			Help:      "Result of the provisioning step",
		}, []string{"operation_id", "instance_id", "step_name", "global_account_id", "plan_id"}),
		deprovisioningResultGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "deprovisioning_step_result",
			Help:      "Result of the deprovisioning step",
		}, []string{"operation_id", "instance_id", "step_name", "global_account_id", "plan_id"}),
	}
}

func (c *StepResultCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningResultGauge.Describe(ch)
	c.deprovisioningResultGauge.Describe(ch)
}

func (c *StepResultCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningResultGauge.Collect(ch)
	c.deprovisioningResultGauge.Collect(ch)
}

func (c *StepResultCollector) OnProvisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.ProvisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected ProvisioningStepProcessed but got %+v", ev)
	}

	var resultValue float64
	switch {
	case stepProcessed.Operation.State == domain.Succeeded:
		resultValue = resultSucceeded
	case stepProcessed.When > 0 && stepProcessed.Error == nil:
		resultValue = resultInProgress
	case stepProcessed.When == 0 && stepProcessed.Error == nil:
		resultValue = resultSucceeded
	case stepProcessed.Error != nil:
		resultValue = resultFailed
	}
	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.provisioningResultGauge.WithLabelValues(
		op.ID,
		op.InstanceID,
		stepProcessed.StepName,
		pp.ErsContext.GlobalAccountID,
		pp.PlanID).Set(resultValue)

	return nil
}

func (c *StepResultCollector) OnDeprovisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.DeprovisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected DeprovisioningStepProcessed but got %+v", ev)
	}

	var resultValue float64
	switch {
	case stepProcessed.When > 0 && stepProcessed.Error == nil:
		resultValue = resultInProgress
	case stepProcessed.When == 0 && stepProcessed.Error == nil:
		resultValue = resultSucceeded
	case stepProcessed.Error != nil:
		resultValue = resultFailed
	}

	// Create_Runtime step always returns operation, 1 second, nil if everything is ok
	// this code is a workaround and should be removed when the step engine is refactored
	if stepProcessed.StepName == "Create_Runtime" && stepProcessed.When == time.Second {
		resultValue = resultSucceeded
	}

	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	c.deprovisioningResultGauge.WithLabelValues(
		op.ID,
		op.InstanceID,
		stepProcessed.StepName,
		pp.ErsContext.GlobalAccountID,
		pp.PlanID).Set(resultValue)
	return nil
}

func (c *StepResultCollector) OnOperationStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.OperationStepProcessed)
	if !ok {
		return fmt.Errorf("expected OperationStepProcessed but got %+v", ev)
	}

	switch {
	case stepProcessed.Operation.Type == "provisioning":
		err := c.OnProvisioningStepProcessed(ctx, ev)
		if err != nil {
			return err
		}
	case stepProcessed.Operation.Type == "deprovisioning":
		err := c.OnDeprovisioningStepProcessed(ctx, ev)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("expected OperationStep of types [provisioning, deprovisioning] but got %+v", stepProcessed.Operation.Type)
	}

	return nil
}
