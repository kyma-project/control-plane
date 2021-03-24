package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	instanceID  = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
)

func TestSkipForTrialPlanStepShouldSkip(t *testing.T) {
	// Given
	log := logrus.New()
	wantSkipTime := time.Duration(0)
	wantOperation := fixOperationWithPlanID(broker.TrialPlanID)

	mockStep := new(automock.Step)
	mockStep.On("Name").Return("Test")
	skipStep := NewSkipForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(wantOperation, log)

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation, gotOperation)
}

func TestSkipForTrialPlanStepShouldNotSkip(t *testing.T) {
	// Given
	log := logrus.New()
	wantSkipTime := time.Duration(10)
	givenOperation1 := fixOperationWithPlanID("operation1")
	wantOperation2 := fixOperationWithPlanID("operation2")

	mockStep := new(automock.Step)
	mockStep.On("Run", givenOperation1, log).Return(wantOperation2, wantSkipTime, nil)
	skipStep := NewSkipForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(givenOperation1, log)

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation2, gotOperation)
}

func fixOperationWithPlanID(planID string) internal.UpgradeKymaOperation {
	upgradeOperation := fixture.FixUpgradeKymaOperation(operationID, instanceID)
	upgradeOperation.ProvisioningParameters = fixture.FixProvisioningParameters("dummy")
	upgradeOperation.ProvisioningParameters.PlanID = planID

	return upgradeOperation
}
