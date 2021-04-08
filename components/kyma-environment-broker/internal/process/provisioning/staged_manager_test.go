package provisioning_test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func TestHappyPath(t *testing.T) {
	// given
	const opID = "op-0001234"
	operation := provisioning.FixProvisionOperation("op-0001234")
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
	operation := provisioning.FixProvisionOperation("op-0001234")
	mgr, operationStorage, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "second", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &testingStep{name: "third", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &onceRetryingStep{name: "first-2", eventPublisher: eventCollector})
	mgr.AddStep("stage-2", &testingStep{name: "second-2", eventPublisher: eventCollector})

	// when
	retry, _ := mgr.Execute(operation.ID)

	// then
	assert.NotZero(t, retry)
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "third", "first-2"})
	op, _ := operationStorage.GetProvisioningOperationByID(operation.ID)
	assert.True(t, op.IsStageFinished("stage-1"))
	assert.False(t, op.IsStageFinished("stage-2"))
}

func TestSkipFinishedStage(t *testing.T) {
	// given
	operation := provisioning.FixProvisionOperation("op-0001234")
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

func TestSkipSucceededSteps(t *testing.T) {
	// given
	operation := provisioning.FixProvisionOperation("op-0001234")

	mgr, _, eventCollector := SetupStagedManager(operation)
	mgr.AddStep("stage-1", &testingStep{name: "first", eventPublisher: eventCollector})
	mgr.AddStep("stage-1", &onceRetryingStep{name: "second", eventPublisher: eventCollector})
	retry, _ := mgr.Execute(operation.ID)
	require.NotZero(t, retry)
	eventCollector.AssertProcessedSteps(t, []string{"first", "second"})

	// when
	retry, _ = mgr.Execute(operation.ID)
	assert.Zero(t, retry)

	// then
	// we expect only one more event as the second round runs only the step which failed before
	eventCollector.AssertProcessedSteps(t, []string{"first", "second", "second"})
}

func SetupStagedManager(op internal.ProvisioningOperation) (*provisioning.StagedManager, storage.Operations, *provisioning.CollectingEventHandler) {
	memoryStorage := storage.NewMemoryStorage()
	memoryStorage.Operations().InsertProvisioningOperation(op)

	eventCollector := &provisioning.CollectingEventHandler{}
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	mgr := provisioning.NewStagedManager(memoryStorage.Operations(), eventCollector, l)
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
