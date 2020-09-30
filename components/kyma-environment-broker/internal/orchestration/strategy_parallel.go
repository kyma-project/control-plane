package orchestration

import (
	"context"
	"sort"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

type ParallelOrchestrationStrategy struct {
	operationExecutor process.Executor
	log               logrus.FieldLogger
}

// TODO(upgrade): Finish implementation and write tests; unused for now
func NewParallelOrchestrationStrategy(operationExecutor process.Executor, log logrus.FieldLogger) Strategy {
	return &ParallelOrchestrationStrategy{
		operationExecutor: operationExecutor,
		log:               log,
	}
}

func (p *ParallelOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec internal.StrategySpec) (time.Duration, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := process.NewQueue(p.operationExecutor, p.log)

	q.Run(ctx.Done(), strategySpec.Parallel.Workers)

	isMaintenanceWindowMode := strategySpec.Schedule == internal.MaintenanceWindow

	// Sort operations according to TimeDelta(Now, internal.RuntimeOperation.MaintenanceWindowBegin)
	if isMaintenanceWindowMode {
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	for _, op := range operations {
		if isMaintenanceWindowMode {
			time.Sleep(time.Until(op.MaintenanceWindowEnd))
		}
		q.Add(op.ID)
	}
	q.ShutDown()

	return 0, nil
}
