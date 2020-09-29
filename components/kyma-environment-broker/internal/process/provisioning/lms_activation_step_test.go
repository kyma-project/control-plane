package provisioning

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

//go:generate mockery -name=Step -output=automock -outpkg=automock -case=underscore

func TestLmsActivationStepShouldNotActivate(t *testing.T) {

	// Given
	memoryStorage := storage.NewMemoryStorage()
	cfg := lms.Config{EnabledForGlobalAccounts: "none"}
	log := logrus.New()
	operation := fixOperationWithPlanID(t, broker.TrialPlanID)
	var activationTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	activationStep := NewLmsActivationStep(memoryStorage.Operations(), cfg, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, operation, returnedOperation)
}

func TestLmsActivationStepShouldActivateForAll(t *testing.T) {

	// Given
	memoryStorage := storage.NewMemoryStorage()
	cfg := lms.Config{EnabledForGlobalAccounts: "all"}
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "another")
	anotherOperation := fixOperationWithPlanID(t, "activated")
	var activationTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, activationTime, nil)

	activationStep := NewLmsActivationStep(memoryStorage.Operations(), cfg, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}

func TestLmsActivationStepShouldActivateForOne(t *testing.T) {

	// Given
	memoryStorage := storage.NewMemoryStorage()
	cfg := lms.Config{EnabledForGlobalAccounts: globalAccountID}
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "another")
	anotherOperation := fixOperationWithPlanID(t, "activated")
	var activationTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, activationTime, nil)

	activationStep := NewLmsActivationStep(memoryStorage.Operations(), cfg, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}
