package provisioning

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

func TestEnableForTrialPlanStepShouldEnable(t *testing.T) {
	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, broker.GcpTrialPlanID)
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator
	anotherOperation := fixOperationWithPlanID(t, "enabled")
	var runTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")
	mockStep.On("Run", operation, log).Return(anotherOperation, runTime, nil)

	enableStep := NewEnableForTrialPlanStep(memoryStorage.Operations(), mockStep)

	// When
	returnedOperation, time, err := enableStep.Run(operation, log)

	// Then
	simpleInputCreator.AssertEnabledComponent(t, mockStep.Name())
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, runTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}

func TestEnableForTrialPlanStepShouldNotEnable(t *testing.T) {
	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "another")
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator
	anotherOperation := fixOperationWithPlanID(t, "not enabled")
	var runTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")
	mockStep.On("Run", operation, log).Return(anotherOperation, runTime, nil)

	enableStep := NewEnableForTrialPlanStep(memoryStorage.Operations(), mockStep)

	// When
	returnedOperation, time, err := enableStep.Run(operation, log)

	// Then
	assert.Empty(t, simpleInputCreator.enabledComponents)
	require.NoError(t, err)
	assert.Equal(t, runTime, time)
	assert.Equal(t, operation, returnedOperation)
}

