package metrics

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/prometheus/client_golang/prometheus"
)

func RegisterAll(sub event.Subscriber, operationStatsGetter OperationsStatsGetter, instanceStatsGetter InstancesStatsGetter) {
	opResultCollector := NewOperationResultCollector()
	opDurationCollector := NewOperationDurationCollector()
	stepResultCollector := NewStepResultCollector()
	prometheus.MustRegister(opResultCollector, opDurationCollector, stepResultCollector)
	prometheus.MustRegister(NewOperationsCollector(operationStatsGetter))
	prometheus.MustRegister(NewInstancesCollector(instanceStatsGetter))

	sub.Subscribe(process.ProvisioningStepProcessed{}, opResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.UpgradeKymaStepProcessed{}, opResultCollector.OnUpgradeKymaStepProcessed)
	sub.Subscribe(process.UpgradeClusterStepProcessed{}, opResultCollector.OnUpgradeClusterStepProcessed)
	sub.Subscribe(process.ProvisioningSucceeded{}, opResultCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.ProvisioningSucceeded{}, opDurationCollector.OnProvisioningSucceeded)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, opDurationCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.ProvisioningStepProcessed{}, stepResultCollector.OnProvisioningStepProcessed)
	sub.Subscribe(process.DeprovisioningStepProcessed{}, stepResultCollector.OnDeprovisioningStepProcessed)
	sub.Subscribe(process.OperationStepProcessed{}, stepResultCollector.OnOperationStepProcessed)
	sub.Subscribe(process.OperationSucceeded{}, opResultCollector.OnOperationSucceeded)
	sub.Subscribe(process.OperationSucceeded{}, opDurationCollector.OnOperationSucceeded)
}
