package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

//go:generate mockery -name=Step -output=automock -outpkg=automock -case=underscore

func TestSkipStepShouldSkip(t *testing.T) {

	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, broker.TrialPlanID)
	var skipTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	skipStep := NewSkipStep(memoryStorage.Operations(), mockStep)

	// When
	returnedOperation, time, err := skipStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, skipTime, time)
	assert.Equal(t, operation, returnedOperation)
}

func TestSkipStepShouldNotSkip(t *testing.T) {

	// Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	operation := fixOperationWithPlanID(t, "another")
	anotherOperation := fixOperationWithPlanID(t, "not skipped")
	var skipTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, skipTime, nil)

	skipStep := NewSkipStep(memoryStorage.Operations(), mockStep)

	// When
	returnedOperation, time, err := skipStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, skipTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)

}

func fixOperationWithPlanID(t *testing.T, planID string) internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: internal.Operation{
			ID:          operationID,
			InstanceID:  instanceID,
			Description: "",
			UpdatedAt:   time.Now(),
		},
		ProvisioningParameters: fixProvisioningParametersWithPlanID(t, planID),
	}
}
