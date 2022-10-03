package process_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	globalAccountID = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
	subAccountID    = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
)

func TestHappyPath(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-2", &testingStep{name: "first-2", eventPublisher: eventCollector}, nil)

	// when
	mgr.Execute(operation.ID)

	// then
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "third", "first-2"})
	op, _ := operationStorage.GetOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func TestHappyPathWithStepCondition(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector}, func(_ internal.Operation) bool {
		return false
	})
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector}, func(_ internal.Operation) bool {
		return true
	})
	mgr.AddStep("stage-2", &testingStep{name: "first-2", eventPublisher: eventCollector}, nil)

	// when
	mgr.Execute(operation.ID)

	// then
	eventCollector.AssertProcessedSteps(t, []string{"first", "third", "first-2"})
	op, _ := operationStorage.GetOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func TestWithRetry(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-2", &onceRetryingStep{name: "first-2", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-2", &testingStep{name: "second-2", eventPublisher: eventCollector}, nil)

	// when
	retry, _ := mgr.Execute(operation.ID)

	// then
	assert.Zero(t, retry)
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "third", "first-2", "first-2", "second-2"})
	op, _ := operationStorage.GetOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func TestSkipFinishedStage(t *testing.T) {
	// given
	operation := FixOperation("op-0001234")
	operation.FinishStage("stage-1")

	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector}, nil)
	mgr.AddStep("stage-2", &testingStep{name: "first-2", eventPublisher: eventCollector}, nil)

	// when
	retry, _ := mgr.Execute(operation.ID)

	// then
	assert.Zero(t, retry)
	eventCollector.WaitForEvents(t, 1)
	op, _ := operationStorage.GetOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.True(t, op.IsStageFinished("stage-2"))
}

func SetupStagedManager(op internal.Operation) (*process.StagedManager, storage.Operations, *CollectingEventHandler) {
	memoryStorage := storage.NewMemoryStorage()
	memoryStorage.Operations().InsertOperation(op)

	eventCollector := &CollectingEventHandler{}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	mgr := process.NewStagedManager(memoryStorage.Operations(), eventCollector, 3*time.Second, l)
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
func (s *testingStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
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
func (s *onceRetryingStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	s.eventPublisher.Publish(context.Background(), s.name)
	if !s.processed {
		s.processed = true
		return operation, time.Millisecond, nil
	}
	logger.Infof("Running")
	return operation, 0, nil
}

func fixProvisioningParametersWithPlanID(planID, region string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:    planID,
		ServiceID: "",
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
		},
		Parameters: internal.ProvisioningParametersDTO{
			Region: ptr.String(region),
			Name:   "dummy",
			Zones:  []string{"europe-west3-b", "europe-west3-c"},
		},
	}
}

func FixOperation(ID string) internal.Operation {
	operation := fixture.FixOperation(ID, "fea2c1a1-139d-43f6-910a-a618828a79d5", internal.OperationTypeProvision)
	operation.FinishedStages = make([]string, 0)
	operation.State = domain.InProgress
	operation.Description = ""
	operation.ProvisioningParameters = fixProvisioningParametersWithPlanID(broker.AzurePlanID, "westeurope")
	return operation
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
	h.StepsProcessed = append(h.StepsProcessed, ev.(process.OperationStepProcessed).StepName)
	return nil
}

func (h *CollectingEventHandler) Publish(ctx context.Context, ev interface{}) {
	switch ev.(type) {
	case process.OperationStepProcessed:
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

type resultCollector struct {
	duration float64
	state    domain.LastOperationState
}

func (rc *resultCollector) OnOperationSucceed(ctx context.Context, ev interface{}) error {
	operationSucceeded, ok := ev.(process.OperationSucceeded)
	if !ok {
		return fmt.Errorf("expected process.OperationSucceeded but got %+v", ev)
	}
	op := operationSucceeded.Operation
	minutes := op.UpdatedAt.Sub(op.CreatedAt).Minutes()
	rc.duration = minutes
	rc.state = op.State
	return nil
}

func (rc *resultCollector) WaitForState(t *testing.T, state domain.LastOperationState) {
	assert.NoError(t, wait.PollImmediate(time.Millisecond, 5*time.Second, func() (bool, error) {
		return rc.state == state, nil
	}))
}

func (rc *resultCollector) AssertSucceededState(t *testing.T) error {
	assert.Equal(t, domain.Succeeded, rc.state)
	return nil
}

func (rc *resultCollector) AssertDurationGreaterThanZero(t *testing.T) error {
	assert.Greater(t, rc.duration, 0.0)
	return nil
}

func SetupStagedManager2(op internal.Operation) (*process.StagedManager, storage.Operations, *event.PubSub) {
	memoryStorage := storage.NewMemoryStorage()
	memoryStorage.Operations().InsertOperation(op)

	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	pubSub := event.NewPubSub(nil)
	mgr := process.NewStagedManager(memoryStorage.Operations(), pubSub, 3*time.Second, l)
	mgr.SpeedUp(100000)
	mgr.DefineStages([]string{"stage-1", "stage-2"})

	return mgr, memoryStorage.Operations(), pubSub
}

func TestOperationSucceededEvent(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := FixOperation("op-0001234")
	mgr, _, pubSub := SetupStagedManager2(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: pubSub}, nil)

	rc := &resultCollector{}
	rc.duration = 123
	pubSub.Subscribe(process.OperationSucceeded{}, rc.OnOperationSucceed)
	fmt.Printf("rc: %.4f \n", rc.duration)

	// when
	mgr.Execute(operation.ID)

	// then
	rc.WaitForState(t, domain.Succeeded)
	rc.AssertDurationGreaterThanZero(t)
}
