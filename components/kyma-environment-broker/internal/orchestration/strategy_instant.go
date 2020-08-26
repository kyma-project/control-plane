package orchestration

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

type InstantOrchestrationStrategy struct {
	executor process.Executor
	log      logrus.FieldLogger
}

// NewInstantOrchestrationStrategy is a simple implementation of the orchestration strategy which just injects all operations into queue
func NewInstantOrchestrationStrategy(executor process.Executor, log logrus.FieldLogger) Strategy {
	return &InstantOrchestrationStrategy{
		executor: executor,
		log:      log,
	}
}

// TODO(upgrade): write tests with UpgradeKymaManager (executor) injected to strategy
func (p *InstantOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec internal.StrategySpec) (time.Duration, error) {
	if len(operations) == 0 {
		return 0, nil
	}
	ctx := context.Background()

	q := process.NewQueue(p.executor, p.log)
	q.Run(ctx.Done(), strategySpec.Parallel.Workers)
	for _, op := range operations {
		q.Add(op.OperationID)
	}
	// shutdown will terminate queue as soon as every key will be processed
	q.ShutDown()

	return 0, nil
}
