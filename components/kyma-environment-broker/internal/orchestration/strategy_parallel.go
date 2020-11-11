package orchestration

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

type ParallelOrchestrationStrategy struct {
	executor process.Executor
	log      logrus.FieldLogger
	wg       map[string]*sync.WaitGroup
	mux      sync.RWMutex
}

// NewParallelOrchestrationStrategy returns a new parallel orchestration strategy, which
// executes operations in parallel using a pool of workers and a delaying queue to support time-based scheduling.
func NewParallelOrchestrationStrategy(executor process.Executor, log logrus.FieldLogger) Strategy {
	return &ParallelOrchestrationStrategy{
		executor: executor,
		log:      log,
		wg:       map[string]*sync.WaitGroup{},
	}
}

// Execute starts the parallel execution of operations.
func (p *ParallelOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec orchestration.StrategySpec) (string, error) {
	if len(operations) == 0 {
		return "", nil
	}
	dq := workqueue.NewDelayingQueue()
	ops := make(chan internal.RuntimeOperation, len(operations))
	execID := uuid.New().String()
	p.mux.Lock()
	defer p.mux.Unlock()
	p.wg[execID] = &sync.WaitGroup{}

	if strategySpec.Schedule == orchestration.MaintenanceWindow {
		sort.Slice(operations, func(i, j int) bool {
			return operations[i].MaintenanceWindowBegin.Before(operations[j].MaintenanceWindowBegin)
		})
	}

	// Create workers
	for i := 0; i < strategySpec.Parallel.Workers; i++ {
		p.createWorker(execID, ops, dq, strategySpec)
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

func (p *ParallelOrchestrationStrategy) createWorker(execID string, ops <-chan internal.RuntimeOperation, dq workqueue.DelayingInterface, strategy orchestration.StrategySpec) {
	p.wg[execID].Add(1)
	go func() {
		for op := range ops {
			p.processOperation(op, dq, strategy)
		}
		p.mux.RLock()
		p.wg[execID].Done()
		p.mux.RUnlock()
	}()
}

func (p *ParallelOrchestrationStrategy) processOperation(op internal.RuntimeOperation, dq workqueue.DelayingInterface, strategy orchestration.StrategySpec) {
	exit := false
	id := op.ID
	log := p.log.WithField("operationID", id)

	switch strategy.Schedule {
	case orchestration.MaintenanceWindow:
		until := time.Until(op.MaintenanceWindowBegin)
		log.Infof("Upgrade operation will be scheduled in %v", until)
		dq.AddAfter(id, until)
	case orchestration.Immediate:
		log.Infof("Upgrade operation is scheduled now")
		dq.Add(id)
	}

	for !exit {
		exit = func() bool {
			key, quit := dq.Get()
			if quit {
				return true
			}
			id := key.(string)
			log = log.WithField("operationID", id)
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("panic error from process: %v", err)
				}
				dq.Done(key)
			}()

			when, err := p.executor.Execute(id)
			if err == nil && when != 0 {
				log.Infof("Adding %q item after %s", id, when)
				dq.AddAfter(key, when)
				return false
			}
			if err != nil {
				log.Errorf("Error from process: %v", err)
			}

			return true
		}()
	}
}
