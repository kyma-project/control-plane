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

	isMaintenanceWindowSchedule := strategySpec.Schedule == internal.MaintenanceWindow

	if isMaintenanceWindowSchedule {
		for i, op := range operations {
			operations[i].MaintenanceWindowBegin = p.resolveWindowBeginTime(op.MaintenanceWindowBegin, op.MaintenanceWindowEnd)
		}
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	for _, op := range operations {
		if isMaintenanceWindowSchedule {
			until := time.Until(op.MaintenanceWindowBegin)
			p.log.Infof("Waiting %v to start upgrade operation %s", until, op.Operation.ID)
			time.Sleep(until)
		}
		q.Add(op.ID)
	}

	return 0, nil
}

// resolves WindowBeginTime to schedule upgrade in the next occurrence of the time window
func (p *ParallelOrchestrationStrategy) resolveWindowBeginTime(beginTime, endTime time.Time) time.Time {
	n := time.Now()
	start := time.Date(n.Year(), n.Month(), n.Day(), beginTime.Hour(), beginTime.Minute(), beginTime.Second(), beginTime.Nanosecond(), beginTime.Location())
	end := time.Date(n.Year(), n.Month(), n.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(), endTime.Location())

	// if time window has already passed we wait until next day
	if start.Before(n) && end.Before(n) {
		start = start.AddDate(0, 0, 1)
	}

	return start
}
