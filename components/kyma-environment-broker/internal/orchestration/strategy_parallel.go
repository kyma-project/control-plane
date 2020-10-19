package orchestration

import (
	"sort"
	"time"

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

func (p *ParallelOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec internal.StrategySpec) (time.Duration, error) {
	if len(operations) == 0 {
		return 0, nil
	}

	stopCh := make(chan struct{})

	q := process.NewQueue(p.executor, p.log)
	q.Run(stopCh, strategySpec.Parallel.Workers)

	isMaintenanceWindowSchedule := strategySpec.Schedule == internal.MaintenanceWindow

	if isMaintenanceWindowSchedule {
		for i, op := range operations {
			operations[i].MaintenanceWindowBegin = p.resolveWindowTime(op.MaintenanceWindowBegin, op.MaintenanceWindowEnd)
		}
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	for _, op := range operations {
		until := time.Until(op.MaintenanceWindowBegin)
		p.log.Infof("Upgrade operation %s will be scheduled in %v", until, op.Operation.ID)
		q.AddAfter(op.ID, until)
	}

	return 0, nil
}

// resolves when is the next occurrence of the time window
func (p *ParallelOrchestrationStrategy) resolveWindowTime(beginTime, endTime time.Time) time.Time {
	n := time.Now()
	start := time.Date(n.Year(), n.Month(), n.Day(), beginTime.Hour(), beginTime.Minute(), beginTime.Second(), beginTime.Nanosecond(), beginTime.Location())
	end := time.Date(n.Year(), n.Month(), n.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), endTime.Nanosecond(), endTime.Location())

	// if time window has already passed we wait until next day
	if start.Before(n) && end.Before(n) {
		start = start.AddDate(0, 0, 1)
	}

	return start
}
