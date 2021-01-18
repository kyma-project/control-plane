package strategies

import (
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type ParallelOrchestrationStrategy struct {
	executor Executor
	dq       map[string]workqueue.DelayingInterface
	wg       map[string]*sync.WaitGroup
	mux      sync.RWMutex
	log      logrus.FieldLogger
}

// NewParallelOrchestrationStrategy returns a new parallel orchestration strategy, which
// executes operations in parallel using a pool of workers and a delaying queue to support time-based scheduling.
func NewParallelOrchestrationStrategy(executor Executor, log logrus.FieldLogger) orchestration.Strategy {
	return &ParallelOrchestrationStrategy{
		executor: executor,
		dq:       map[string]workqueue.DelayingInterface{},
		wg:       map[string]*sync.WaitGroup{},
		log:      log,
	}
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

	// Create workers
	for i := 0; i < strategySpec.Parallel.Workers; i++ {
		p.createWorker(execID, ops, strategySpec)
	}

	// Send operations to workers
	for _, op := range operations {
		ops <- op
	}
	close(ops)

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
	p.mux.Lock()
	defer p.mux.Unlock()
	p.log.Infof("Cancelling strategy execution %s", executionID)
	p.dq[executionID].ShutDown()
}

func (p *ParallelOrchestrationStrategy) createWorker(execID string, ops <-chan orchestration.RuntimeOperation, strategy orchestration.StrategySpec) {
	p.wg[execID].Add(1)
	go func() {
		for op := range ops {
			err := p.processOperation(op, strategy, execID)
			if err != nil {
				p.log.Errorf("while processing operation %s: %v", op.ID, err)
			}
		}
		p.mux.RLock()
		p.wg[execID].Done()
		p.mux.RUnlock()
	}()
}

func (p *ParallelOrchestrationStrategy) processOperation(op orchestration.RuntimeOperation, strategy orchestration.StrategySpec, executionID string) error {
	exit := false
	id := op.ID
	log := p.log.WithField("operationID", id)

	switch strategy.Schedule {
	case orchestration.MaintenanceWindow:
		until := time.Until(op.MaintenanceWindowBegin)
		log.Infof("Upgrade operation will be scheduled in %v", until)
		p.dq[executionID].AddAfter(id, until)
	case orchestration.Immediate:
		log.Infof("Upgrade operation is scheduled now")
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
