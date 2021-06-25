package deprovisioning

import (
	"testing"
	"time"

	hyperscalerMocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	fixOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	fixRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	fixGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	fixProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"
)

func TestInitialisationStep_Run(t *testing.T) {
	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", mock.Anything, mock.AnythingOfType("string")).Return(nil)

	t.Run("Should mark operation as Succeeded when operation has succeeded", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		operation := fixDeprovisioningOperation()
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		assert.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: nil,
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, accountProviderMock, nil, time.Hour)

		// when
		operation, repeat, err := step.Run(operation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, operation.State)

		storedOp, err := memoryStorage.Operations().GetDeprovisioningOperationByID(operation.ID)
		assert.Equal(t, operation, *storedOp)
		assert.NoError(t, err)

		_, err = memoryStorage.Instances().GetByID(instance.InstanceID)
		assert.True(t, dberr.IsNotFound(err))
	})

	t.Run("Should delete instance and userID when operation has succeeded", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

		operation := fixDeprovisioningOperation()
		operation.ProvisionerOperationID = ""
		operation.State = domain.Succeeded
		err := memoryStorage.Operations().InsertDeprovisioningOperation(operation)
		assert.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		assert.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		instance.RuntimeID = ""
		err = memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, accountProviderMock, nil, time.Hour)

		// when
		operation, repeat, err := step.Run(operation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, operation.State)
		assert.Equal(t, "", operation.ProvisioningParameters.ErsContext.UserID)

		inst, err := memoryStorage.Instances().GetByID(operation.InstanceID)
		assert.Error(t, err)
		assert.Nil(t, inst)

		storedOp, err := memoryStorage.Operations().GetDeprovisioningOperationByID(operation.ID)
		assert.NoError(t, err)
		assert.Equal(t, operation, *storedOp)
	})

}

func fixDeprovisioningOperation() internal.DeprovisioningOperation {
	deprovisioniningOperation := fixture.FixDeprovisioningOperation(fixOperationID, fixInstanceID)
	return deprovisioniningOperation
}

func fixProvisioningOperation() internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(fixOperationID, fixInstanceID)
	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(fixInstanceID)
	instance.RuntimeID = fixRuntimeID
	instance.GlobalAccountID = fixGlobalAccountID

	return instance
}
