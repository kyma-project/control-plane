package provisioning

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

type StagedManager struct {
	log              logrus.FieldLogger
	operationStorage storage.Operations
	publisher        event.Publisher

	steps  map[string]map[int][]Step
	stages []string
}

func NewStagedManager(storage storage.Operations, pub event.Publisher, logger logrus.FieldLogger) *StagedManager {
	return &StagedManager{
		log:              logger,
		operationStorage: storage,
		publisher:        pub,
		stages:           []string{},
		steps:            make(map[string]map[int][]Step, 0),
	}
}

func (m *StagedManager) AddStep(stage string, weight int, step Step) {
	if _, exists := m.steps[stage]; !exists {
		m.steps[stage] = make(map[int][]Step)
		m.stages = append(m.stages, stage)
	}
	m.steps[stage][weight] = append(m.steps[stage][weight], step)
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
		if processedOperation.IsStageFinished(stage) {
			continue
		}
		stageRetry := 0 * time.Second
		for _, weightStep := range m.sortWeight(stage) {
			steps := m.steps[stage][weightStep]
			for _, step := range steps {
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

				if when == 0 {
					// mark step processed
					operation.FinishStep(step.Name())
				}

				// the step needs a retry
				if when > 0 {
					// remember to stageRetry
					if stageRetry == 0 {
						stageRetry = when
					}
					if when < stageRetry {
						stageRetry = when
					}
				}
			}
			if stageRetry > 0 {
				return stageRetry, nil
			}
		}

		if stageRetry == 0 {
			processedOperation.FinishStage(stage)
			op, err := m.operationStorage.UpdateProvisioningOperation(processedOperation)
			if err != nil {
				logOperation.Infof("Unable to save operation with finished stage %s: %s", stage, err.Error())
				return time.Second, nil
			}
			logOperation.Infof("Finished stage %s", stage)
			processedOperation = *op
		}

	}

	return 0, nil
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

func (m *StagedManager) sortWeight(stage string) []int {
	var weight []int
	for w := range m.steps[stage] {
		weight = append(weight, w)
	}
	sort.Ints(weight)

	return weight
}
