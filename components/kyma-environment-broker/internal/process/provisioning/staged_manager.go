package provisioning

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	"time"

	"github.com/sirupsen/logrus"
)

type StagedManager struct {
	log              logrus.FieldLogger
	operationStorage storage.Operations
	publisher        event.Publisher

	stages           []*stage
	operationTimeout time.Duration

	mu sync.RWMutex
	finishedSteps    map[string]struct{}
}

type Step interface {
	Name() string
	Run(operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error)
}

type stage struct {
	name  string
	steps []Step
}

func (s *stage) AddStep(step Step) {
	s.steps = append(s.steps, step)
}

func NewStagedManager(storage storage.Operations, pub event.Publisher, operationTimeout time.Duration, logger logrus.FieldLogger) *StagedManager {
	return &StagedManager{
		log:              logger,
		operationStorage: storage,
		publisher:        pub,
		operationTimeout: operationTimeout,
	}
}

func (m *StagedManager) DefineStages(names []string) {
	m.stages = make([]*stage, len(names))
	for i, n := range names {
		m.stages[i] = &stage{name: n, steps: []Step{}}
	}
}

func (m *StagedManager) AddStep(stageName string, step Step) error {
	for _, s := range m.stages {
		if s.name == stageName {
			s.AddStep(step)
			return nil
		}
	}
	return fmt.Errorf("Stage %s not defined", stageName)
}

func (m *StagedManager) GetAllSteps() []Step {
	var allSteps []Step
	for _, s := range m.stages {
		allSteps = append(allSteps, s.steps...)
	}
	return allSteps
}

func (m *StagedManager) Execute(operationID string) (time.Duration, error) {
	operation, err := m.operationStorage.GetProvisioningOperationByID(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch operation from storage: %s", err)
		return 3 * time.Second, nil
	}

	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID, "planID": operation.ProvisioningParameters.PlanID})
	logOperation.Infof("Start process operation steps for GlobalAcocunt=%s, ", operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if time.Since(operation.CreatedAt) > m.operationTimeout {
		logOperation.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		operation.State = domain.Failed
		_, err = m.operationStorage.UpdateProvisioningOperation(*operation)
		if err != nil {
			logOperation.Infof("Unable to save operation with finished the provisioning process")
			return time.Second, err
		}
		return 0, errors.New("operation has reached the time limit")
	}

	var when time.Duration
	processedOperation := *operation

	for _, stage := range m.stages {
		if processedOperation.IsStageFinished(stage.name) {
			continue
		}

		for _, step := range stage.steps {
			if m.IsStepDone(operation, step) {
				continue
			}
			logStep := logOperation.WithField("step", step.Name()).
				WithField("stage", stage.name)
			logStep.Infof("Start step")

			processedOperation, when, err = m.runStep(step, processedOperation, logStep)
			if err != nil {
				logStep.Errorf("Process operation failed: %s", err)
				return 0, err
			}
			if processedOperation.State != domain.InProgress {
				logStep.Infof("Operation %q got status %s. Process finished.", operation.ID, processedOperation.State)
				return 0, nil
			}

			// the step needs a retry
			if when > 0 {
				return when, nil
			}

			// mark step processed
			m.finishStep(operation, step)
		}

		processedOperation, err = m.saveFinishedStage(processedOperation, stage, logOperation)
		if err != nil {
			return time.Second, nil
		}
	}

	processedOperation.State = domain.Succeeded
	_, err = m.operationStorage.UpdateProvisioningOperation(processedOperation)
	if err != nil {
		logOperation.Infof("Unable to save operation with finished the provisioning process")
		return time.Second, err
	}

	return 0, nil
}

func (m *StagedManager) saveFinishedStage(operation internal.ProvisioningOperation, s *stage, log logrus.FieldLogger) (internal.ProvisioningOperation, error) {
	operation.FinishStage(s.name)
	op, err := m.operationStorage.UpdateProvisioningOperation(operation)
	if err != nil {
		log.Infof("Unable to save operation with finished stage %s: %s", s.name, err.Error())
		return operation, err
	}
	log.Infof("Finished stage %s", s.name)
	return *op, nil
}

func (m *StagedManager) runStep(step Step, operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	start := time.Now()
	processedOperation, when, err := step.Run(operation, logger)
	m.publisher.Publish(context.TODO(), process.ProvisioningStepProcessed{
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

// TODO: input builder is not cached!!!!

func (m *StagedManager) IsStepDone(operation *internal.ProvisioningOperation, step Step) bool {
	///return operation.IsStepDone(step.Name()
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s/%s", operation.ID, step.Name())
	_, found := m.finishedSteps[key]
	return found
}

func (m *StagedManager) finishStep(operation *internal.ProvisioningOperation, step Step) {
	//return operation.FinishStep(step.Name())
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s/%s", operation.ID, step.Name())
	m.finishedSteps[key] = struct{}{}
}
