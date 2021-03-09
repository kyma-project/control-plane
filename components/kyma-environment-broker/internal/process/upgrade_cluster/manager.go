package upgrade_cluster

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Step interface {
	Name() string
	Run(operation internal.UpgradeClusterOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error)
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

func (m *Manager) Execute(operationID string) (time.Duration, error) {
	return 0, nil
}
