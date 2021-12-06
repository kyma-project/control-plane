package strategies

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

type ParallelOrchestrationStrategy struct {
	executor        orchestration.OperationExecutor
	dq              map[string]workqueue.DelayingInterface // scheduling queue, delaying queue for all pending & in progress ops
	pq              map[string]workqueue.DelayingInterface // processing queue, delaying queue for the in progress ops
	wg              map[string]*sync.WaitGroup
	mux             sync.RWMutex
	log             logrus.FieldLogger
	rescheduleDelay time.Duration
	scheduleNum     map[string]int
}

// NewParallelOrchestrationStrategy returns a new parallel orchestration strategy, which
// executes operations in parallel using a pool of workers and a delaying queue to support time-based scheduling.
func NewParallelOrchestrationStrategy(executor orchestration.OperationExecutor, log logrus.FieldLogger, rescheduleDelay time.Duration) orchestration.Strategy {
	strategy := &ParallelOrchestrationStrategy{
		executor:        executor,
		dq:              map[string]workqueue.DelayingInterface{},
		pq:              map[string]workqueue.DelayingInterface{},
		wg:              map[string]*sync.WaitGroup{},
		log:             log,
		rescheduleDelay: rescheduleDelay,
		scheduleNum:     map[string]int{},
	}

	return strategy
}

// Execute starts the parallel execution of operations.
func (p *ParallelOrchestrationStrategy) Execute(operations []orchestration.RuntimeOperation, strategySpec orchestration.StrategySpec) (string, error) {
	if len(operations) == 0 {
		return "", nil
	}

	execID := uuid.New().String()
	p.mux.Lock()
	defer p.mux.Unlock()
	p.scheduleNum[execID] = len(operations)
	p.wg[execID] = &sync.WaitGroup{}
	p.dq[execID] = workqueue.NewDelayingQueue()
	p.pq[execID] = workqueue.NewDelayingQueue()

	for i, op := range operations {
		duration, err := p.updateMaintenanceWindow(execID, &operations[i], strategySpec)
		if err != nil {
			//error when read from storage or update to storage during maintenance window reschedule
			p.handleRescheduleErrorOperation(execID, &operations[i])
			p.log.Errorf("while processing operation %s: %v, will reschedule it", op.ID, err)
		} else {
			p.dq[execID].AddAfter(&operations[i], duration)
		}
	}

	// Create workers
	for i := 0; i < strategySpec.Parallel.Workers; i++ {
		p.createWorker(execID, strategySpec)
	}

	return execID, nil
}

func (p *ParallelOrchestrationStrategy) createWorker(execID string, strategy orchestration.StrategySpec) {
	p.wg[execID].Add(1)

	go func() {
		p.scheduleOperationsLoop(execID, strategy)

		p.mux.RLock()
		p.wg[execID].Done()
		p.mux.RUnlock()
	}()
}

func (p *ParallelOrchestrationStrategy) scheduleOperationsLoop(execID string, strategy orchestration.StrategySpec) {
	p.mux.RLock()
	dq := p.dq[execID]
	pq := p.pq[execID]
	p.mux.RUnlock()

	for {
		p.mux.RLock()
		if p.scheduleNum[execID] <= 0 {
			dq.ShutDown()
			pq.ShutDown()
		}
		p.mux.RUnlock()

		item, shutdown := dq.Get()
		if shutdown {
			p.log.Infof("scheduling queue is shutdown")
			break
		}

		op := item.(*orchestration.RuntimeOperation)

		// check the window before process for the case if op Get is not in time
		duration, err := p.updateMaintenanceWindow(execID, op, strategy)
		if err != nil {
			//error when read from storage or update to storage
			p.handleRescheduleErrorOperation(execID, op)
			dq.Done(item)
			continue
		}

		log := p.log.WithField("operationID", op.ID)
		if duration <= 0 {
			log.Infof("operation is scheduled now")

			pq.Add(item)
			p.processOperation(execID)

			p.mux.Lock()
			p.scheduleNum[execID]--
			p.mux.Unlock()
		} else {
			log.Infof("operation will be scheduled in %v", duration)
			dq.AddAfter(item, duration)
			dq.Done(item)
		}

	}
}

func (p *ParallelOrchestrationStrategy) processOperation(execID string) {
	exit := false

	for !exit {
		exit = func() bool {
			item, quit := p.pq[execID].Get()
			if quit {
				p.log.Infof("processing queue is shutdown")
				return true
			}

			op := item.(*orchestration.RuntimeOperation)
			id := op.ID
			log := p.log.WithField("operationID", id)

			defer func() {
				if err := recover(); err != nil {
					log.Errorf("panic error from process: %v. Stacktrace: %s", err, debug.Stack())
				}
				p.pq[execID].Done(item)
			}()

			when, err := p.executor.Execute(id)
			if err == nil && when != 0 {
				log.Infof("Adding %q item after %v", id, when)
				p.pq[execID].AddAfter(item, when)
				return false
			}
			if err != nil {
				log.Errorf("Error from process: %v", err)
			}

			log.Infof("Finishing processing operation")
			p.dq[execID].Done(item)

			return true
		}()
	}

}

func (p *ParallelOrchestrationStrategy) updateMaintenanceWindow(execID string, op *orchestration.RuntimeOperation, strategy orchestration.StrategySpec) (time.Duration, error) {
	var duration time.Duration
	id := op.ID

	switch strategy.Schedule {
	case orchestration.MaintenanceWindow:
		// if time window for this operation has finished, we requeue and reprocess on next time window
		if !op.MaintenanceWindowEnd.IsZero() && op.MaintenanceWindowEnd.Before(time.Now()) {
			if p.rescheduleDelay > 0 {
				op.MaintenanceWindowBegin = op.MaintenanceWindowBegin.Add(p.rescheduleDelay)
				op.MaintenanceWindowEnd = op.MaintenanceWindowEnd.Add(p.rescheduleDelay)
			} else {
				currentDay := op.MaintenanceWindowBegin.Weekday()
				diff := orchestration.NextAvailableDayDiff(currentDay, orchestration.ConvertSliceOfDaysToMap(op.MaintenanceDays))
				op.MaintenanceWindowBegin = op.MaintenanceWindowBegin.AddDate(0, 0, diff)
				op.MaintenanceWindowEnd = op.MaintenanceWindowEnd.AddDate(0, 0, diff)
			}

			err := p.executor.Reschedule(id, op.MaintenanceWindowBegin, op.MaintenanceWindowEnd)
			//error when read from storage or update to storage
			if err != nil {
				errors.Wrap(err, "while rescheduling operation by executor (still continuing with new schedule)")
				return duration, err
			}
		}

		duration = time.Until(op.MaintenanceWindowBegin)

	case orchestration.Immediate:
	}

	return duration, nil
}

func (p *ParallelOrchestrationStrategy) Wait(executionID string) {
	p.mux.RLock()
	wg := p.wg[executionID]
	p.mux.RUnlock()
	if wg != nil {
		wg.Wait()
	}
}

func (p *ParallelOrchestrationStrategy) Cancel(executionID string) {
	if executionID == "" {
		return
	}
	p.log.Infof("Cancelling strategy execution %s", executionID)

	p.mux.Lock()
	defer p.mux.Unlock()
	dq := p.dq[executionID]
	pq := p.pq[executionID]

	if dq != nil {
		dq.ShutDown()
	}

	if pq != nil {
		pq.ShutDown()
	}
}

func (p *ParallelOrchestrationStrategy) handleRescheduleErrorOperation(execID string, op *orchestration.RuntimeOperation) {
	p.dq[execID].AddAfter(op, 24*time.Hour)
}
