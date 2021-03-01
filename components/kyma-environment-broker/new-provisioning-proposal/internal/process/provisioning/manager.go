package provisioning

import (
	"fmt"
	"sort"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/storage"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Step interface {
	Name() string
	Run(operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error)
}

type Manager struct {
	log              logrus.FieldLogger
	steps            map[int][]Step
	operationStorage storage.Operations
	stepTimeout      time.Duration
}

func NewManager(storage storage.Operations, logger logrus.FieldLogger) *Manager {
	return &Manager{
		log:              logger,
		operationStorage: storage,
		steps:            make(map[int][]Step, 0),
		stepTimeout:      15 * time.Second,
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

func (m *Manager) runStep(step Step, operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	processedOperation, when, err := step.Run(operation, logger)

	return processedOperation, when, err
}

func (m *Manager) Execute(operationID string) error {
	operation, err := m.getOperation(operationID)
	if err != nil {
		m.log.Errorf("Cannot fetch operation from storage: %s", err)
		return err
	}

	var when time.Duration
	processedOperation := *operation
	logOperation := m.log.WithFields(logrus.Fields{"operation": operationID, "instanceID": operation.InstanceID})

	logOperation.Info("Start process operation steps")
	for _, weightStep := range m.sortWeight() {
		steps := m.steps[weightStep]
		for _, step := range steps {
			now := time.Now()
			exit := false
			for !exit {
				logStep := logOperation.WithField("step", step.Name())
				logStep.Infof("Start step")

				if time.Now().Sub(now) > m.stepTimeout {
					logStep.Errorf("timeout for step")
					return fmt.Errorf("step %s has timed out ", step.Name())
				}

				processedOperation, when, err = m.runStep(step, processedOperation, logStep)
				if err != nil {
					logStep.Errorf("Process operation failed: %s", err)
					processedOperation.State = domain.Failed
					m.updateOperation(processedOperation)
					return err
				}
				if processedOperation.State != domain.InProgress {
					logStep.Infof("Operation %q got status %s. Process finished.", operation.ID, processedOperation.State)
					return nil
				}
				if when == 0 {
					logStep.Info("Process operation successful")
					exit = true
					continue
				}

				logStep.Infof("Process operation will be repeated in %s ...", when)
				time.Sleep(when)
			}
			m.log.Info("")
		}
	}

	processedOperation.State = domain.Succeeded
	m.updateOperation(processedOperation)

	logOperation.Infof("Operation %q got status %s. All steps finished.", operation.ID, processedOperation.State)
	return nil
}

func (m *Manager) getOperation(operationID string) (*internal.ProvisioningOperation, error) {
	var operation *internal.ProvisioningOperation
	err := wait.PollImmediate(3*time.Second, time.Minute, func() (done bool, err error) {
		op, err := m.operationStorage.GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		operation = op
		return true, nil
	})
	return operation, err
}

func (m *Manager) updateOperation(operation internal.ProvisioningOperation) {
	err := wait.PollImmediate(10*time.Second, time.Minute, func() (done bool, err error) {
		_, err = m.operationStorage.UpdateProvisioningOperation(operation)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		m.log.Errorf("cannot update operation: %s", err)
	}
}

func (m *Manager) sortWeight() []int {
	var weight []int
	for w := range m.steps {
		weight = append(weight, w)
	}
	sort.Ints(weight)

	return weight
}
