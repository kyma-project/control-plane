package provisioning

import (
	"context"
	"testing"
	"time"

	hyperscalerautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	azuretesting "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure/testing"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AzureResourceGroupHappyPath(t *testing.T) {
	// given
	tags := fixTags()
	memoryStorage := storage.NewMemoryStorage()
	accountProvider := fixAccountProvider()
	azureClient := azuretesting.NewFakeAzureClientHappyPath()
	step := fixAzureResourceGroupStep(memoryStorage.Operations(), azuretesting.NewFakeHyperscalerProvider(azureClient), accountProvider)
	inputCreator := newInputCreator()
	op := fixOperationAzureResourceGroup(inputCreator)
	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := memoryStorage.Operations().InsertProvisioningOperation(op)
	require.NoError(t, err)

	// when
	op, when, err := step.Run(op, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, when)
	inputCreator.AssertNoOverrides(t)
	assert.NotEmpty(t, op.Azure.ResourceGroupName)
	assert.Equal(t, azureClient.Tags, tags)

	// when retrying completed step
	op, when, err = step.Run(op, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, when)
}

func Test_AzureResourceGroupUnhappyPath(t *testing.T) {
	tests := []struct {
		name                string
		giveOperation       func(t *testing.T) internal.ProvisioningOperation
		giveStep            func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureResourceGroupStep
		wantRepeatOperation bool
	}{
		{
			name:          "Provision parameter errors",
			giveOperation: fixInvalidProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureResourceGroupStep {
				accountProvider := fixAccountProvider()
				return *fixAzureResourceGroupStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: false,
		},
		{
			name:          "AccountProvider cannot get gardener credentials",
			giveOperation: fixOperationAzureResourceGroupSimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureResourceGroupStep {
				accountProvider := fixAccountProviderGardenerCredentialsError()
				return *fixAzureResourceGroupStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while getting config from HAP",
			giveOperation: fixOperationAzureResourceGroupSimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureResourceGroupStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureResourceGroupStep(storage.Operations(),
					azuretesting.NewFakeHyperscalerProviderError(),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: false,
		},
		{
			name:          "Error while creating Azure ResourceGroup",
			giveOperation: fixOperationAzureResourceGroupSimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureResourceGroupStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureResourceGroupStep(storage.Operations(),
					// ups ... resource group cannot be created
					azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientResourceGroupError()),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			memoryStorage := storage.NewMemoryStorage()
			op := tt.giveOperation(t)
			step := tt.giveStep(t, memoryStorage)
			// this is required to avoid storage retries (without this statement there will be an error => retry)
			err := memoryStorage.Operations().InsertProvisioningOperation(op)
			require.NoError(t, err)

			// when
			op.UpdatedAt = time.Now()
			op, when, err := step.Run(op, fixLogger())
			require.NotNil(t, op)

			// then
			if tt.wantRepeatOperation {
				ensureOperationIsRepeated(t, err, when)
			} else {
				ensureOperationIsNotRepeated(t, err)
			}
		})
	}
}
func fixAzureResourceGroupStep(memoryStorageOp storage.Operations, hyperscalerProvider azure.HyperscalerProvider,
	accountProvider *hyperscalerautomock.AccountProvider) *ProvisionAzureResourceGroupStep {
	return NewProvisionAzureResourceGroupStep(memoryStorageOp, hyperscalerProvider, accountProvider, context.Background())
}

func fixOperationAzureResourceGroupSimple(t *testing.T) internal.ProvisioningOperation {
	return fixOperationAzureResourceGroup(newInputCreator())
}

func fixOperationAzureResourceGroup(inputCreator *simpleInputCreator) internal.ProvisioningOperation {
	op := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ID:         fixOperationID,
			InstanceID: fixInstanceID,
		},
		ProvisioningParameters: `{
			"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
			"ers_context": {
				"subaccount_id": "` + fixSubAccountID + `"
			},
			"parameters": {
				"name": "nachtmaar-15",
				"components": [],
				"region": "westeurope"
			}
		}`,
		InputCreator: inputCreator,
	}
	return op
}
