package strategies

import (
	"sort"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

type Executor interface {
	Execute(operationID string) (time.Duration, error)
}

type ParallelOrchestrationStrategy struct {
	executor        Executor
	dq              workqueue.DelayingInterface
	orchestrations  storage.Orchestrations
	orchestrationID string
	pollingInterval time.Duration
	wg              map[string]*sync.WaitGroup
	mux             sync.RWMutex
	log             logrus.FieldLogger
}

// NewParallelOrchestrationStrategy returns a new parallel orchestration strategy, which
// executes operations in parallel using a pool of workers and a delaying queue to support time-based scheduling.
func NewParallelOrchestrationStrategy(executor Executor, orchestrations storage.Orchestrations, orchestrationID string, pollingInterval time.Duration, log logrus.FieldLogger) orchestration.Strategy {
	return &ParallelOrchestrationStrategy{
		executor:        executor,
		orchestrations:  orchestrations,
		orchestrationID: orchestrationID,
		pollingInterval: pollingInterval,
		dq:              workqueue.NewDelayingQueue(),
		wg:              map[string]*sync.WaitGroup{},
		log:             log,
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

func (p *ParallelOrchestrationStrategy) createWorker(execID string, ops <-chan orchestration.RuntimeOperation, strategy orchestration.StrategySpec) {
	p.wg[execID].Add(1)
	go func() {
		for op := range ops {
			err := p.processOperation(op, strategy)
			if err != nil {
				p.log.Errorf("while processing operation %s: %v", op.ID, err)
			}
		}
		p.mux.RLock()
		p.wg[execID].Done()
		p.mux.RUnlock()
	}()
}

func (p *ParallelOrchestrationStrategy) processOperation(op orchestration.RuntimeOperation, strategy orchestration.StrategySpec) error {
	exit := false
	until := time.Second
	id := op.ID
	log := p.log.WithField("operationID", id)

	if strategy.Schedule == orchestration.MaintenanceWindow {
		until = time.Until(op.MaintenanceWindowBegin)
		log.Infof("Upgrade operation will be scheduled in %v", until)
	}
	isCanceled, err := p.checkIfOrchestrationIsCanceled(until, log)
	if err != nil {
		return errors.Wrap(err, "while checking if orchestration was canceled")
	}
	if isCanceled {
		log.Info("Operation was canceled")
		return nil
	}
	log.Infof("Upgrade operation is scheduled now")
	p.dq.Add(id)

	for !exit {
		exit = func() bool {
			key, quit := p.dq.Get()
			if quit {
				return true
			}
			id := key.(string)
			log = log.WithField("operationID", id)
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("panic error from process: %v", err)
				}
				p.dq.Done(key)
			}()

			when, err := p.executor.Execute(id)
			if err == nil && when != 0 {
				log.Infof("Adding %q item after %s", id, when)
				p.dq.AddAfter(key, when)
				return false
			}
			if err != nil {
				log.Errorf("Error from process: %v", err)
			}

			return true
		}()
	}
	return nil
}

func (p *ParallelOrchestrationStrategy) checkIfOrchestrationIsCanceled(until time.Duration, log logrus.FieldLogger) (bool, error) {
	isCanceled := false
	var lastErr error
	err := wait.PollImmediate(p.pollingInterval, until, func() (bool, error) {
		log.Infof("Checking if orchestration was canceled")
		o, err := p.orchestrations.GetByID(p.orchestrationID)
		if err != nil {
			if dberr.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
		if o.State == orchestration.Canceled {
			isCanceled = true
			return true, nil
		}
		return false, nil
	})
	if err != nil && err != wait.ErrWaitTimeout {
		return false, errors.Wrap(lastErr, "while polling operations")
	}
	return isCanceled, nil
}
