package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestEnableForTrialPlanStepShouldSkip(t *testing.T) {
	// Given
	log := logrus.New()
	wantSkipTime := time.Duration(0)
	wantOperation := fixOperationWithPlanID("non-trial-plan")

	mockStep := new(automock.Step)
	mockStep.On("Name").Return("Test")
	enableStep := NewEnableForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := enableStep.Run(wantOperation, log)

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation, gotOperation)
}

func TestEnableForTrialPlanStepShouldNotSkip(t *testing.T) {
	// Given
	log := logrus.New()
	wantSkipTime := time.Duration(10)
	givenOperation1 := fixOperationWithPlanID(broker.TrialPlanID)
	wantOperation2 := fixOperationWithPlanID("operation2")

	mockStep := new(automock.Step)
	mockStep.On("Run", givenOperation1, log).Return(wantOperation2, wantSkipTime, nil)
	skipStep := NewEnableForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(givenOperation1, log)

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation2, gotOperation)
}

func fixOperationWithPlanID(planID string) internal.Operation {
	Operation := fixture.FixProvisioningOperation(operationID, instanceID)
	Operation.ProvisioningParameters = fixProvisioningParametersWithPlanID(planID, "region")

	return Operation
}

func fixOperationWithPlanIDAndKymaVersion(planID, version string) internal.Operation {
	Operation := fixOperationWithPlanID(planID)
	Operation.RuntimeVersion.Version = version

	return Operation
}
