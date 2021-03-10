package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate mockery -name=Step -output=automock -outpkg=automock -case=underscore

func TestSkipForTrialPlanStepShouldSkip(t *testing.T) {

	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID(broker.TrialPlanID)
	var skipTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	skipStep := NewAzureEventHubActivationStep(mockStep)

	// When
	returnedOperation, time, err := skipStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, skipTime, time)
	assert.Equal(t, operation, returnedOperation)
}

func TestSkipForTrialPlanStepShouldNotSkip(t *testing.T) {

	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID("another")
	anotherOperation := fixOperationWithPlanID("not skipped")
	var skipTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, skipTime, nil)

	skipStep := NewAzureEventHubActivationStep(mockStep)

	// When
	returnedOperation, time, err := skipStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, skipTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)

}

func fixOperationWithPlanID(planID string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	provisioningOperation.ProvisioningParameters = fixProvisioningParametersWithPlanID(planID, "region")

	return provisioningOperation
}
