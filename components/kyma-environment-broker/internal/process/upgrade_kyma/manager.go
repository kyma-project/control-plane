package upgrade_kyma

import (
	"context"
	"sort"
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

func (m *Manager) Execute(operationID string) (time.Duration, error) {
	op, err := m.operationStorage.GetUpgradeKymaOperationByID(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch operation from storage: %s", err)
		return 3 * time.Second, nil
	}
	operation := *op
	if operation.IsFinished() {
		return 0, nil
	}

	var when time.Duration
	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID})

	logOperation.Info("Start process operation steps")
	for _, weightStep := range m.sortWeight() {
		steps := m.steps[weightStep]
		for _, step := range steps {
			logStep := logOperation.WithField("step", step.Name())
			logStep.Infof("Start step")

			operation, when, err = m.runStep(step, operation, logStep)
			if err != nil {
				logStep.Errorf("Process operation failed: %s", err)
				return 0, err
			}
			if operation.IsFinished() {
				logStep.Infof("Operation %q got status %s. Process finished.", operation.Operation.ID, operation.State)
				return 0, nil
			}
			if when == 0 {
				logStep.Info("Process operation successful")
				continue
			}

			logStep.Infof("Process operation will be repeated in %s ...", when)
			return when, nil
		}
	}

	logOperation.Infof("Operation %q got status %s. All steps finished.", operation.Operation.ID, operation.State)
	return 0, nil
}

func (m *Manager) sortWeight() []int {
	var weight []int
	for w := range m.steps {
		weight = append(weight, w)
	}
	sort.Ints(weight)

	return weight
}
