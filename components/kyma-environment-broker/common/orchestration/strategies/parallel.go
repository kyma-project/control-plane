package strategies

import (
	"runtime/debug"
	"sort"
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
	dq              map[string]workqueue.DelayingInterface
	wg              map[string]*sync.WaitGroup
	mux             sync.RWMutex
	log             logrus.FieldLogger
	rescheduleDelay time.Duration
}

// NewParallelOrchestrationStrategy returns a new parallel orchestration strategy, which
// executes operations in parallel using a pool of workers and a delaying queue to support time-based scheduling.
func NewParallelOrchestrationStrategy(executor orchestration.OperationExecutor, log logrus.FieldLogger, rescheduleDelay time.Duration) orchestration.Strategy {
	strategy := &ParallelOrchestrationStrategy{
		executor:        executor,
		dq:              map[string]workqueue.DelayingInterface{},
		wg:              map[string]*sync.WaitGroup{},
		log:             log,
		rescheduleDelay: rescheduleDelay,
	}
	if strategy.rescheduleDelay <= 0 {
		strategy.rescheduleDelay = 24 * time.Hour
	}
	return strategy
}

// Execute starts the parallel execution of operations.
func (p *ParallelOrchestrationStrategy) Execute(operations []orchestration.RuntimeOperation, strategySpec orchestration.StrategySpec) (string, error) {
	if len(operations) == 0 {
		return "", nil
	}
	ops := make(chan orchestration.RuntimeOperation, len(operations))
	execID := uuid.New().String()
	p.mux.Lock()
	defer p.mux.Unlock()
	p.wg[execID] = &sync.WaitGroup{}
	p.dq[execID] = workqueue.NewDelayingQueue()

	if strategySpec.Schedule == orchestration.MaintenanceWindow {
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	// Send operations to workers
	for _, op := range operations {
		ops <- op
	}

	// Create workers
	for i := 0; i < strategySpec.Parallel.Workers; i++ {
		p.createWorker(execID, ops, strategySpec)
	}

	return execID, nil
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
	if dq != nil {
		dq.ShutDown()
	}
}

func (p *ParallelOrchestrationStrategy) createWorker(execID string, ops chan orchestration.RuntimeOperation, strategy orchestration.StrategySpec) {
	p.wg[execID].Add(1)
	go func() {
		moreOperations := true
		for moreOperations {
			select {
			case op := <-ops:
				err := p.processOperation(op, ops, strategy, execID)
				if err != nil {
					p.log.Errorf("while processing operation %s: %v", op.ID, err)
				}
			default:
				p.log.Infof("Idle worker for %s exiting", execID)
				moreOperations = false
			}
		}
		p.mux.RLock()
		p.wg[execID].Done()
		p.mux.RUnlock()
	}()
}

func (p *ParallelOrchestrationStrategy) processOperation(op orchestration.RuntimeOperation, ops chan orchestration.RuntimeOperation, strategy orchestration.StrategySpec, executionID string) error {
	exit := false
	id := op.ID
	log := p.log.WithField("operationID", id)

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
			if err != nil {
				errors.Wrap(err, "while rescheduling operation by executor (still continuing with new schedule)")
			}
			ops <- op
			log.Infof("operation will be rescheduled starting at %v", op.MaintenanceWindowBegin)
			return err
		}

		until := time.Until(op.MaintenanceWindowBegin)
		log.Infof("operation will be scheduled in %v", until)
		p.dq[executionID].AddAfter(id, until)
	case orchestration.Immediate:
		log.Infof("operation is scheduled now")
		p.dq[executionID].Add(id)
	}

	for !exit {
		exit = func() bool {
			key, quit := p.dq[executionID].Get()
			if quit {
				return true
			}
			id := key.(string)
			log = log.WithField("operationID", id)
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("panic error from process: %v. Stacktrace: %s", err, debug.Stack())
				}
				p.dq[executionID].Done(key)
			}()

			when, err := p.executor.Execute(id)
			if err == nil && when != 0 {
				log.Infof("Adding %q item after %s", id, when)
				p.dq[executionID].AddAfter(key, when)
				return false
			}
			if err != nil {
				log.Errorf("Error from process: %v", err)
			}
			return true
		}()
	}
	log.Info("Finishing processing operation")
	return nil
}
