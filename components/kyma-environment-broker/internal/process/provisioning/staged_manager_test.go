package provisioning_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHappyPath(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixProvisionOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &testingStep{name: "first-2", eventPublisher: eventCollector})

	// when
	mgr.Execute(operation.ID)

	// then
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "third", "first-2"})
	op, _ := operationStorage.GetProvisioningOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func TestWithRetry(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixProvisionOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &onceRetryingStep{name: "first-2", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &testingStep{name: "second-2", eventPublisher: eventCollector})

	// when
	retry, _ := mgr.Execute(operation.ID)

	// then
	assert.Zero(t, retry)
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "third", "first-2", "first-2", "second-2"})
	op, _ := operationStorage.GetProvisioningOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func TestSkipFinishedStage(t *testing.T) {
	// given
	operation := FixProvisionOperation("op-0001234")
	operation.FinishStage("stage-1")

	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &testingStep{name: "first-2", eventPublisher: eventCollector})

	// when
	retry, _ := mgr.Execute(operation.ID)

	// then
	assert.Zero(t, retry)
	eventCollector.WaitForEvents(t, 1)
	op, _ := operationStorage.GetProvisioningOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func SetupStagedManager(op internal.ProvisioningOperation) (*provisioning.StagedManager, storage.Operations, *CollectingEventHandler) {
	memoryStorage := storage.NewMemoryStorage()
	memoryStorage.Operations().InsertProvisioningOperation(op)

	eventCollector := &CollectingEventHandler{}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	mgr := provisioning.NewStagedManager(memoryStorage.Operations(), eventCollector, 3*time.Second, l)
	mgr.SpeedUp(100000)
	mgr.DefineStages([]string{"stage-1", "stage-2"})

	return mgr, memoryStorage.Operations(), eventCollector
}

type testingStep struct {
	name           string
	eventPublisher event.Publisher
}

func (s *testingStep) Name() string {
	return s.name
}
func (s *testingStep) Run(operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	logger.Infof("Running")
	s.eventPublisher.Publish(context.Background(), s.name)
	return operation, 0, nil
}

type onceRetryingStep struct {
	name           string
	processed      bool
	eventPublisher event.Publisher
}

func (s *onceRetryingStep) Name() string {
	return s.name
}
func (s *onceRetryingStep) Run(operation internal.ProvisioningOperation, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	s.eventPublisher.Publish(context.Background(), s.name)
	if !s.processed {
		s.processed = true
		return operation, time.Millisecond, nil
	}
	logger.Infof("Running")
	return operation, 0, nil
}

func FixProvisionOperation(ID string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(ID, "fea2c1a1-139d-43f6-910a-a618828a79d5")
	provisioningOperation.FinishedStages = make(map[string]struct{})
	provisioningOperation.State = domain.InProgress
	provisioningOperation.Description = ""
	provisioningOperation.ProvisioningParameters = provisioning.FixProvisioningParameters(broker.AzurePlanID, "westeurope")

	return provisioningOperation
}

type CollectingEventHandler struct {
	mu             sync.Mutex
	StepsProcessed []string // collects events from the Manager
	stepsExecuted  []string // collects events from testing steps
}

func (h *CollectingEventHandler) OnStepExecuted(_ context.Context, ev interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stepsExecuted = append(h.stepsExecuted, ev.(string))
	return nil
}

func (h *CollectingEventHandler) OnStepProcessed(_ context.Context, ev interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.StepsProcessed = append(h.StepsProcessed, ev.(process.ProvisioningStepProcessed).StepName)
	return nil
}

func (h *CollectingEventHandler) Publish(ctx context.Context, ev interface{}) {
	switch ev.(type) {
	case process.ProvisioningStepProcessed:
		h.OnStepProcessed(ctx, ev)
	case string:
		h.OnStepExecuted(ctx, ev)
	}
}

func (h *CollectingEventHandler) WaitForEvents(t *testing.T, count int) {
	assert.NoError(t, wait.PollImmediate(time.Millisecond, time.Second, func() (bool, error) {
		return len(h.StepsProcessed) == count, nil
	}))
}

func (h *CollectingEventHandler) AssertProcessedSteps(t *testing.T, stepNames []string) {
	h.WaitForEvents(t, len(stepNames))
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, stepName := range stepNames {
		processed := h.StepsProcessed[i]
		executed := h.stepsExecuted[i]
		assert.Equal(t, stepName, processed)
		assert.Equal(t, stepName, executed)
	}
	assert.Len(t, h.StepsProcessed, len(stepNames))
}
