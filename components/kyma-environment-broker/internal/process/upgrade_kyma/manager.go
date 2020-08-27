package upgrade_kyma

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Step interface {
	Name() string
	Run(operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error)
}

type Manager struct {
	log              logrus.FieldLogger
	steps            map[int][]Step
	operationStorage storage.Operations

	publisher event.Publisher
}

func NewManager(storage storage.Operations, pub event.Publisher, logger logrus.FieldLogger) *Manager {
	return &Manager{
		log:              logger,
		steps:            make(map[int][]Step, 0),
		operationStorage: storage,
		publisher:        pub,
	}
}

func (m *Manager) InitStep(step Step) {
	m.AddStep(0, step)
}

func (m *Manager) AddStep(weight int, step Step) {
	if weight <= 0 {
		weight = 1
	}
	m.steps[weight] = append(m.steps[weight], step)
}

func (m *Manager) runStep(step Step, operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	start := time.Now()
	processedOperation, when, err := step.Run(operation, logger)
	m.publisher.Publish(context.TODO(), process.UpgradeKymaStepProcessed{
		OldOperation: operation,
		Operation:    processedOperation,
		StepProcessed: process.StepProcessed{
			StepName: step.Name(),
			Duration: time.Since(start),
			When:     when,
			Error:    err,
		},
	})
	return processedOperation, when, err
}
