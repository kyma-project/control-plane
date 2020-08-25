package orchestration

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/sirupsen/logrus"
)

type InstantOrchestrationStrategy struct {
	q   *process.Queue
	log logrus.FieldLogger
}

// NewInstantOrchestrationStrategy is a simple implementation of the orchestration strategy which just injects all operations into queue
func NewInstantOrchestrationStrategy(queue *process.Queue, log logrus.FieldLogger) Strategy {
	return &InstantOrchestrationStrategy{
		q:   queue,
		log: log,
	}
}

// TODO(upgrade): write tests with UpgradeKymaManager (executor) injected to strategy
func (p *InstantOrchestrationStrategy) Execute(operations []internal.RuntimeOperation, strategySpec internal.StrategySpec) (time.Duration, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.q.Run(ctx.Done(), strategySpec.Parallel.Workers)
	for _, op := range operations {
		p.q.Add(op.OperationID)
	}

	return 0, nil
}
