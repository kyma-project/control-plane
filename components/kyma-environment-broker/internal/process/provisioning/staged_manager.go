package provisioning

import (
	"context"
	"fmt"

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

	stages []*stage
}

type stage struct {
	name  string
	steps []Step
}

type stepInfo struct {
	step   Step
	weight int
}

func (s *stage) AddStep(step Step) {
	s.steps = append(s.steps, step)
}

func NewStagedManager(storage storage.Operations, pub event.Publisher, logger logrus.FieldLogger) *StagedManager {
	return &StagedManager{
		log:              logger,
		operationStorage: storage,
		publisher:        pub,
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

	var when time.Duration
	processedOperation := *operation

	for _, stage := range m.stages {
		if processedOperation.IsStageFinished(stage.name) {
			continue
		}

		for _, step := range stage.steps {
			if operation.IsStepDone(step.Name()) {
				continue
			}
			logStep := logOperation.WithField("step", step.Name()).
				WithField("stage", stage)
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
			operation.FinishStep(step.Name())
		}

		processedOperation, err = m.saveFinishedStage(processedOperation, stage, logOperation)
		if err != nil {
			return time.Second, nil
		}
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
