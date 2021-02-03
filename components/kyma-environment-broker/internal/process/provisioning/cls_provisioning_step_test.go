package provisioning

import (
	"testing"
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