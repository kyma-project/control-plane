package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma/automock"
)

//go:generate mockery -name=Step -output=automock -outpkg=automock -case=underscore

const (
	globalAccountID = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
)

func TestLmsActivationStepShouldNotActivate(t *testing.T) {
	// Given
	cfg := lms.Config{EnabledForGlobalAccounts: "none"}
	log := logrus.New()
	operation := fixOperationWithPlanID(broker.TrialPlanID)
	var activationTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	activationStep := NewLmsActivationStep(cfg, mockStep)

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
	cfg := lms.Config{EnabledForGlobalAccounts: "all"}
	log := logrus.New()
	operation := fixOperationWithPlanID("another")
	anotherOperation := fixOperationWithPlanID("activated")
	var activationTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, activationTime, nil)

	activationStep := NewLmsActivationStep(cfg, mockStep)

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

	cfg := lms.Config{EnabledForGlobalAccounts: globalAccountID}
	log := logrus.New()
	operation := fixLMSOperationWithPlanID("another")
	anotherOperation := fixOperationWithPlanID("activated")
	var activationTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, activationTime, nil)

	activationStep := NewLmsActivationStep(cfg, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}

func fixLMSOperationWithPlanID(planID string) internal.UpgradeKymaOperation {
	ersContext := internal.ERSContext{
		TenantID:        "",
		SubAccountID:    subAccountID,
		GlobalAccountID: globalAccountID,
		ServiceManager:  nil,
		Active:          nil,
		UserID:          "",
	}
	upgradeOperation := fixture.FixUpgradeKymaOperation(operationID, instanceID)
	upgradeOperation.ProvisioningParameters = fixture.FixProvisioningParameters("dummy")
	upgradeOperation.ProvisioningParameters.PlanID = planID
	upgradeOperation.ProvisioningParameters.ErsContext = ersContext

	return upgradeOperation
}
