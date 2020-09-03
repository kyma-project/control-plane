package upgrade_kyma
//
//import (
//	"testing"
//	"time"
//
//	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
//	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
//	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
//	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
//	"github.com/stretchr/testify/assert"
//)
//
//const (
//	kymaVersion            = "1.10"
//	globalAccountID        = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
//	subAccountID           = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
//	instanceID             = "58f8c703-1756-48ab-9299-a847974d1fee"
//	operationID            = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
//)
//
//func TestUpgradeKymaStep_Run(t *testing.T) {
//	// given
//	//log := logrus.New()
//	memoryStorage := storage.NewMemoryStorage()
//
//	operation := fixOperationUpgradeKyma()
//	err := memoryStorage.Operations().InsertUpgradeKymaOperation(operation)
//	assert.NoError(t, err)
//
//	provisioningOperation := fixProvisioningOperation()
//	err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
//	assert.NoError(t, err)
//
//	provisionerClient := provisionerAutomock.Client{}
//	provisionerClient.On("UpgradeRuntime", globalAccountID, subAccountID, gqlschema.UpgradeRuntimeInput{
//		KymaConfig: &gqlschema.KymaConfigInput{
//			Version:       "",
//			Components:    nil,
//			Configuration: nil,
//	}})
//}
//
//func fixOperationUpgradeKyma() internal.UpgradeKymaOperation {
//	return internal.UpgradeKymaOperation{
//		Operation: internal.Operation{
//			ID:          operationID,
//			InstanceID:  instanceID,
//			Description: "",
//			UpdatedAt:   time.Now(),
//		},
//	}
//}
//
//func fixProvisioningOperation() internal.ProvisioningOperation {
//	return internal.ProvisioningOperation{
//		Operation: internal.Operation{
//			ID:                     "0b3b7a4a-3ea0-49fc-b09e-85306d8ac5b8",
//			InstanceID:             instanceID,
//			ProvisionerOperationID: "b3285fa4-6ccc-4c52-9a56-6884115eb97a",
//			Description:            "",
//			UpdatedAt:              time.Now(),
//		},
//		ProvisioningParameters: `{"ers_context":{"globalaccount_id":"1"}}`,
//	}
//}