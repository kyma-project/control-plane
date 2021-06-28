package update

import (
	"errors"
	"fmt"
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"

	"time"

	"github.com/sirupsen/logrus"
)

type Manager struct {
	log              logrus.FieldLogger
	operationStorage storage.Operations
	publisher        event.Publisher

	stages           []*stage
	operationTimeout time.Duration

	mu sync.RWMutex

	speedFactor int64
}

type Step interface {
	Name() string
	Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error)
}

type stage struct {
	name  string
	steps []Step
}

func (s *stage) AddStep(step Step) {
	s.steps = append(s.steps, step)
}

func NewManager(storage storage.Operations, pub event.Publisher, operationTimeout time.Duration, logger logrus.FieldLogger) *Manager {
	return &Manager{
		log:              logger,
		operationStorage: storage,
		publisher:        pub,
		operationTimeout: operationTimeout,
		speedFactor:      1,
	}
}

// SpeedUp changes speedFactor parameter to reduce the sleep time if a step needs a retry.
// This method should only be used for testing purposes
func (m *Manager) SpeedUp(speedFactor int64) {
	m.speedFactor = speedFactor
}

func (m *Manager) DefineStages(names []string) {
	m.stages = make([]*stage, len(names))
	for i, n := range names {
		m.stages[i] = &stage{name: n, steps: []Step{}}
	}
}

func (m *Manager) AddStep(stageName string, step Step) error {
	for _, s := range m.stages {
		if s.name == stageName {
			s.AddStep(step)
			return nil
		}
	}
	return fmt.Errorf("Stage %s not defined", stageName)
}

func (m *Manager) GetAllStages() []string {
	var all []string
	for _, s := range m.stages {
		all = append(all, s.name)
	}
	return all
}

func (m *Manager) Execute(operationID string) (time.Duration, error) {
	operation, err := m.operationStorage.GetUpdatingOperationByID(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch operation from storage: %s", err)
		return 3 * time.Second, nil
	}

	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID, "planID": operation.ProvisioningParameters.PlanID})
	logOperation.Infof("Start process operation steps for GlobalAcocunt=%s, ", operation.ProvisioningParameters.ErsContext.GlobalAccountID)
	if time.Since(operation.CreatedAt) > m.operationTimeout {
		logOperation.Infof("operation has reached the time limit: operation was created at: %s", operation.CreatedAt)
		operation.State = domain.Failed
		_, err = m.operationStorage.UpdateUpdatingOperation(*operation)
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

			logStep := logOperation.WithField("step", step.Name()).
				WithField("stage", stage.name)
			logStep.Infof("Start step")

			processedOperation, when, err = m.runStep(step, processedOperation, logStep)
			if err != nil {
				logStep.Errorf("Process operation failed: %s", err)
				return 0, err
			}
			if processedOperation.State == domain.Failed || processedOperation.State == domain.Succeeded {
				logStep.Infof("Operation %q got status %s. Process finished.", operation.ID, processedOperation.State)
				return 0, nil
			}

			// the step needs a retry
			if when > 0 {
				return when, nil
			}
		}

		processedOperation, err = m.saveFinishedStage(processedOperation, stage, logOperation)
		if err != nil {
			return time.Second, nil
		}
	}

	processedOperation.State = domain.Succeeded
	_, err = m.operationStorage.UpdateUpdatingOperation(processedOperation)
	if err != nil {
		logOperation.Infof("Unable to save operation with finished the provisioning process")
		return time.Second, err
	}

	return 0, nil
}

func (m *Manager) saveFinishedStage(operation internal.UpdatingOperation, s *stage, log logrus.FieldLogger) (internal.UpdatingOperation, error) {
	operation.FinishStage(s.name)
	op, err := m.operationStorage.UpdateUpdatingOperation(operation)
	if err != nil {
		log.Infof("Unable to save operation with finished stage %s: %s", s.name, err.Error())
		return operation, err
	}
	log.Infof("Finished stage %s", s.name)
	return *op, nil
}

func (m *Manager) runStep(step Step, operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	begin := time.Now()
	for {
		//start := time.Now()
		processedOperation, when, err := step.Run(operation, logger)
		//m.publisher.Publish(context.TODO(), process.UpdateStepProcessed{
		//	OldOperation: operation,
		//	Operation:    processedOperation,
		//	StepProcessed: process.StepProcessed{
		//		StepName: step.Name(),
		//		Duration: time.Since(start),
		//		When:     when,
		//		Error:    err,
		//	},
		//})

		// break the loop if:
		// - the step does not need a retry
		// - step returns an error
		// - the loop takes too much time (to not block the worker too long)
		if when == 0 || err != nil || time.Since(begin) > 5 * time.Minute {
			return processedOperation, when, err
		}
		time.Sleep(when / time.Duration(m.speedFactor))
	}
}
