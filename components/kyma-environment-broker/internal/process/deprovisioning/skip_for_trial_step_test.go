package deprovisioning

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
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
	givenStorage := storage.NewMemoryStorage()
	wantOperation := fixOperationWithPlanID(t, broker.TrialPlanID)

	mockStep := new(automock.Step)
	mockStep.On("Name").Return("Test")
	skipStep := NewSkipForTrialPlanStep(givenStorage.Operations(), mockStep)

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
	givenStorage := storage.NewMemoryStorage()
	givenOperation1 := fixOperationWithPlanID(t, "operation1")
	wantOperation2 := fixOperationWithPlanID(t, "operation2")

	mockStep := new(automock.Step)
	mockStep.On("Run", givenOperation1, log).Return(wantOperation2, wantSkipTime, nil)
	skipStep := NewSkipForTrialPlanStep(givenStorage.Operations(), mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(givenOperation1, log)

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation2, gotOperation)
}

func fixOperationWithPlanID(t *testing.T, planID string) internal.DeprovisioningOperation {
	t.Helper()

	return internal.DeprovisioningOperation{
		Operation: internal.Operation{
			ID:         operationID,
			InstanceID: instanceID,
			UpdatedAt:  time.Now(),
		},
		ProvisioningParameters: fixProvisioningParametersWithPlanID(t, planID),
	}
}

func fixProvisioningParametersWithPlanID(t *testing.T, planID string) string {
	t.Helper()

	parameters := internal.ProvisioningParameters{
		PlanID: planID,
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
			ServiceManager: &internal.ServiceManagerEntryDTO{
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: serviceManagerUser,
						Password: serviceManagerPassword,
					},
				},
				URL: serviceManagerURL,
			},
		},
		Parameters: internal.ProvisioningParametersDTO{
			Name:   "dummy",
			Region: ptr.String("europe-west4-a"),
			Zones:  []string{"europe-west4-b", "europe-west4-c"},
		},
	}

	rawParameters, err := json.Marshal(parameters)
	if err != nil {
		t.Errorf("cannot marshal provisioning parameters: %s", err)
	}

	return string(rawParameters)
}
