package deprovisioning

import (
	"context"
	"sort"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
)

type Step interface {
	Name() string
	Run(operation internal.DeprovisioningOperation, logger logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error)
}

type Manager struct {
	log              logrus.FieldLogger
	steps            map[int][]Step
	operationStorage storage.Operations
	operationManager *process.DeprovisionOperationManager

	publisher event.Publisher
}

func NewManager(storage storage.Operations, pub event.Publisher, logger logrus.FieldLogger) *Manager {
	return &Manager{
		log:              logger,
		operationStorage: storage,
		steps:            make(map[int][]Step, 0),
		publisher:        pub,
		operationManager: process.NewDeprovisionOperationManager(storage),
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

func (m *Manager) runStep(step Step, operation internal.DeprovisioningOperation, logger logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	start := time.Now()
	processedOperation, when, err := step.Run(operation, logger)
	if err != nil {
		processedOperation.LastError = kebError.ReasonForError(err)
		log := logger.WithFields(logrus.Fields{"error_component": processedOperation.LastError.Component(), "error_reason": processedOperation.LastError.Reason()})
		log.Errorf("Last error: %s", processedOperation.LastError.Error())
		// no saving to storage for deprovisioning
	}

	m.publisher.Publish(context.TODO(), process.DeprovisioningStepProcessed{
		StepProcessed: process.StepProcessed{
			StepName: step.Name(),
			Duration: time.Since(start),
			When:     when,
			Error:    err,
		},
		OldOperation: operation,
		Operation:    processedOperation,
	})
	return processedOperation, when, err
}

func (m *Manager) Execute(operationID string) (time.Duration, error) {
	op, err := m.operationStorage.GetDeprovisioningOperationByID(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch DeprovisioningOperation from storage: %s", err)
		return 3 * time.Second, nil
	}
	operation := *op

	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID})

	provisioningOp, err := m.operationStorage.GetProvisioningOperationByInstanceID(op.InstanceID)
	if err != nil {
		m.log.Errorf("Cannot fetch ProvisioningOperation for instanceID %s from storage: %s", op.InstanceID, err)

		operation.LastError = kebError.ReasonForError(err)
		log := logOperation.WithFields(logrus.Fields{"error_component": operation.LastError.Component(), "error_reason": operation.LastError.Reason()})
		log.Errorf("Last error: %s", operation.LastError.Error())
		m.publisher.Publish(context.TODO(), process.DeprovisioningStepProcessed{
			StepProcessed: process.StepProcessed{
				Error: err,
			},
			Operation: operation,
		})

		_, duration, err := m.operationManager.OperationFailed(operation, "Error retrieving provisioning operation", err, log)

		return duration, err
	}

	logOperation = logOperation.WithField("planID", provisioningOp.ProvisioningParameters.PlanID)

	var when time.Duration
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
			if operation.State != domain.InProgress && operation.State != orchestration.Pending {
				if operation.RuntimeID == "" && operation.State == domain.Succeeded {
					logStep.Infof("Operation %q has no runtime ID. Process finished.", operation.ID)
					return when, nil
				}
				logStep.Infof("Operation %q got status %s. Process finished.", operation.ID, operation.State)
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

	logOperation.Infof("Operation %q got status %s. All steps finished.", operation.ID, operation.State)
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
