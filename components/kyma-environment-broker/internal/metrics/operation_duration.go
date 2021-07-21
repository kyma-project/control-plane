package metrics

import (
	"context"
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/prometheus/client_golang/prometheus"
)

// OperationDurationCollector provides histograms which describes the time of provisioning/deprovisioning operations:
// - compass_keb_provisioning_duration_minutes
// - compass_keb_deprovisioning_duration_minutes
type OperationDurationCollector struct {
	provisioningHistogram   *prometheus.HistogramVec
	deprovisioningHistogram *prometheus.HistogramVec
}

func NewOperationDurationCollector() *OperationDurationCollector {
	return &OperationDurationCollector{
		provisioningHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "provisioning_duration_minutes",
			Help:      "The time of the provisioning process",
			Buckets:   prometheus.LinearBuckets(20, 2, 40),
		}, []string{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}),
		deprovisioningHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "deprovisioning_duration_minutes",
			Help:      "The time of the deprovisioning process",
			Buckets:   prometheus.LinearBuckets(1, 1, 30),
		}, []string{"operation_id", "runtime_id", "instance_id", "global_account_id", "plan_id"}),
	}
}

func (c *OperationDurationCollector) Describe(ch chan<- *prometheus.Desc) {
	c.provisioningHistogram.Describe(ch)
	c.deprovisioningHistogram.Describe(ch)
}

func (c *OperationDurationCollector) Collect(ch chan<- prometheus.Metric) {
	c.provisioningHistogram.Collect(ch)
	c.deprovisioningHistogram.Collect(ch)
}

func (c *OperationDurationCollector) OnProvisioningSucceeded(ctx context.Context, ev interface{}) error {
	provision, ok := ev.(process.ProvisioningSucceeded)
	if !ok {
		return fmt.Errorf("expected process.ProvisioningSucceeded but got %+v", ev)
	}

	op := provision.Operation
	pp := op.ProvisioningParameters
	minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
	c.provisioningHistogram.
		WithLabelValues(op.ID, op.RuntimeID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).Observe(minutes)

	return nil
}

func (c *OperationDurationCollector) OnDeprovisioningStepProcessed(ctx context.Context, ev interface{}) error {
	stepProcessed, ok := ev.(process.DeprovisioningStepProcessed)
	if !ok {
		return fmt.Errorf("expected process.DeprovisioningStepProcessed but got %+v", ev)
	}

	op := stepProcessed.Operation
	pp := op.ProvisioningParameters
	if stepProcessed.OldOperation.State == domain.InProgress && op.State == domain.Succeeded {
		minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
		c.deprovisioningHistogram.
			WithLabelValues(op.ID, op.RuntimeID, op.InstanceID, pp.ErsContext.GlobalAccountID, pp.PlanID).Observe(minutes)
	}

	return nil
}
