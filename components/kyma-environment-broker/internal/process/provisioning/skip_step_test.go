package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
)

//go:generate mockery -name=Step -output=automock -outpkg=automock -case=underscore

func TestSkipStepWillSkip(t *testing.T) {

	//Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	skipID := "test"
	operation := fixOperationWithPlanID(t, skipID)
	var skipTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	skipStep := NewSkipStep(memoryStorage.Operations(), skipID, mockStep)

	//When

	returnedOperation, time, err := skipStep.Run(operation, log)
	//Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, skipTime, time)
	assert.Equal(t, operation, returnedOperation)
}

func TestSkipStepWillNotSkip(t *testing.T) {

	//Given
	memoryStorage := storage.NewMemoryStorage()
	log := logrus.New()
	skipID := "test"
	operation := fixOperationWithPlanID(t, "another")
	anotherOperation := fixOperationWithPlanID(t, "not skipped")
	var skipTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, skipTime, nil)

	skipStep := NewSkipStep(memoryStorage.Operations(), skipID, mockStep)

	//When
	returnedOperation, time, err := skipStep.Run(operation, log)

	//Then
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
