package provisioning

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
)

func TestEnableForTrialPlanStepShouldEnable(t *testing.T) {
	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID(broker.TrialPlanID)
	operation.ProvisioningParameters.Parameters.KymaVersion = "1.20.0"
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator
	anotherOperation := fixOperationWithPlanID("enabled")
	var runTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")
	mockStep.On("Run", operation, log).Return(anotherOperation, runTime, nil)

	enableStep := NewNatsActivationStep(mockStep)

	// When
	returnedOperation, time, err := enableStep.Run(operation, log)

	// Then
	require.NoError(t, err)
	assert.Equal(t, runTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}

func TestEnableForTrialPlanStepShouldNotEnable(t *testing.T) {
	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID("another")
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator
	anotherOperation := fixOperationWithPlanID("not enabled")
	var runTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")
	mockStep.On("Run", operation, log).Return(anotherOperation, runTime, nil)

	enableStep := NewNatsActivationStep(mockStep)

	// When
	returnedOperation, time, err := enableStep.Run(operation, log)

	// Then
	assert.Empty(t, simpleInputCreator.enabledComponents)
	require.NoError(t, err)
	assert.Equal(t, runTime, time)
	assert.Equal(t, operation, returnedOperation)
}
