package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClsProvisioningStep_Run(t *testing.T) {
	// given
	//repo := storage.NewMemoryStorage().Operations()
	//// TODO: Change this to new servicemanager instatiation
	//clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	//clientFactory.SynchronousProvisioning()
	//operation := internal.ProvisioningOperation{
	//	Operation: internal.Operation{
	//		InstanceDetails: internal.InstanceDetails{
	//			Cls: internal.ClsData{Instance: internal.ServiceManagerInstanceInfo{
	//				BrokerID:  "broker-id",
	//				ServiceID: "svc-id",
	//				PlanID:    "plan-id",
	//			}},
	//			ShootDomain: "cls-test.sap.com",
	//		},
	//	},
	//	SMClientFactory: clientFactory,
	//}
	//offeringStep := NewClsOfferingStep()
	//offeringStep := NewClsOfferingStep(repo)

	//provisionStep := NewProvideClsInstaceStep(repo)
	//repo.InsertProvisioningOperation(operation)
	//
	//log := logger.NewLogDummy()
	//// when
	//operation, retry, err := offeringStep.Run(operation, log)
	//require.NoError(t, err)
	//require.Zero(t, retry)
	//
	//operation, retry, err = provisionStep.Run(operation, logger.NewLogDummy())
	//
	//// then
	//assert.NoError(t, err)
	//assert.Zero(t, retry)
	//assert.NotEmpty(t, operation.Cls.Instance.InstanceID)
	//assert.False(t, operation.Cls.Instance.Provisioned)
	//assert.True(t, operation.Cls.Instance.ProvisioningTriggered)
	//clientFactory.AssertProvisionCalled(t, servicemanager.InstanceKey{
	//	BrokerID:   "broker-id",
	//	InstanceID: operation.Cls.Instance.InstanceID,
	//	ServiceID:  "svc-id",
	//	PlanID:     "plan-id",
	//})
}

func TestClsActivationStepShouldActivateForOne(t *testing.T) {
	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID("another")
	anotherOperation := fixOperationWithPlanID("activated")
	var activationTime time.Duration = 10

	mockStep := &automock.Step{}
	mockStep.On("Run", operation, log).Return(anotherOperation, activationTime, nil)

	activationStep := NewClsActivationStep(false, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, anotherOperation, returnedOperation)
}

func TestClsActivationStepShouldNotActivate(t *testing.T) {
	// Given
	log := logrus.New()
	operation := fixOperationWithPlanID(broker.TrialPlanID)
	var activationTime time.Duration = 0

	mockStep := &automock.Step{}
	mockStep.On("Name").Return("Test")

	activationStep := NewClsActivationStep(false, mockStep)

	// When
	returnedOperation, time, err := activationStep.Run(operation, log)

	// Then
	mockStep.AssertExpectations(t)
	require.NoError(t, err)
	assert.Equal(t, activationTime, time)
	assert.Equal(t, operation, returnedOperation)
}