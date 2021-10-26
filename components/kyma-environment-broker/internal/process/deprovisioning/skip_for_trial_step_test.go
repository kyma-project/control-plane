package deprovisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	instanceID      = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID     = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	subAccountID    = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	globalAccountID = "80ac17bd-33e8-4ffa-8d56-1d5367755723"

	serviceManagerURL      = "http://sm.com"
	serviceManagerUser     = "admin"
	serviceManagerPassword = "admin123"
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

func fixOperationWithPlanID(planID string) internal.DeprovisioningOperation {
	deprovisioningOperation := fixture.FixDeprovisioningOperation(operationID, instanceID)
	deprovisioningOperation.Operation.ProvisioningParameters.PlanID = planID
	deprovisioningOperation.Operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
	deprovisioningOperation.Operation.ProvisioningParameters.ErsContext.SubAccountID = subAccountID
	deprovisioningOperation.Operation.ProvisioningParameters.ErsContext.ServiceManager = &internal.ServiceManagerEntryDTO{
		Credentials: internal.ServiceManagerCredentials{
			BasicAuth: internal.ServiceManagerBasicAuth{
				Username: serviceManagerUser,
				Password: serviceManagerPassword,
			},
		},
		URL: serviceManagerURL,
	}
	return deprovisioningOperation
}
