package orchestration

import (
	"sort"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

type ParallelOrchestrationStrategy struct {
	executor process.Executor
	log      logrus.FieldLogger
}

func NewParallelOrchestrationStrategy(executor process.Executor, log logrus.FieldLogger) Strategy {
	return &ParallelOrchestrationStrategy{
		executor: executor,
		log:      log,
	}
}

func (p *ParallelOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec orchestration.StrategySpec) (time.Duration, error) {
	if len(operations) == 0 {
		return 0, nil
	}

	stopCh := make(chan struct{})

	q := process.NewQueue(p.executor, p.log)
	q.Run(stopCh, strategySpec.Parallel.Workers)

	if strategySpec.Schedule == orchestration.MaintenanceWindow {
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	for _, op := range operations {
		switch strategySpec.Schedule {
		case orchestration.MaintenanceWindow:
			until := time.Until(op.MaintenanceWindowBegin)
			p.log.Infof("Upgrade operation %s will be scheduled in %v", op.ID, until)
			q.AddAfter(op.ID, until)
		case orchestration.Immediate:
			q.Add(op.ID)
		}
	}

	return 0, nil
}
