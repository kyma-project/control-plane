package provisioning

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/containerregistry/mgmt/2019-05-01/containerregistry"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	hyperscalerautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	azuretesting "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure/testing"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AzureContainerRegistryHappyPath(t *testing.T) {
	// given
	tags := fixTags()
	memoryStorage := storage.NewMemoryStorage()
	accountProvider := fixAccountProvider()
	azureClient := azuretesting.NewFakeAzureClientHappyPath()
	step := fixAzureContainerRegistryStep(memoryStorage.Operations(), azuretesting.NewFakeHyperscalerProvider(azureClient), accountProvider)
	inputCreator := newInputCreator()
	op := fixOperationAzureContainerRegistry(inputCreator)
	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := memoryStorage.Operations().InsertProvisioningOperation(op)
	require.NoError(t, err)

	// when
	op, when, err := step.Run(op, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, when)
	assert.True(t, op.Azure.ContainerRegistryCreated)
	assert.Equal(t, azureClient.Tags, tags)
	inputCreator.AssertOverride(t, components.Serverless, gqlschema.ConfigEntryInput{Key: "dockerRegistry.enableInternal", Value: "false", Secret: ptr.Bool(true)})
	inputCreator.AssertOverride(t, components.Serverless, gqlschema.ConfigEntryInput{Key: "dockerRegistry.username", Value: "rtestinstanceid", Secret: ptr.Bool(true)})
	inputCreator.AssertOverride(t, components.Serverless, gqlschema.ConfigEntryInput{Key: "dockerRegistry.password", Value: "some-password", Secret: ptr.Bool(true)})
	inputCreator.AssertOverride(t, components.Serverless, gqlschema.ConfigEntryInput{Key: "dockerRegistry.serverAddress", Value: "rtestinstanceid.azurecr.io", Secret: ptr.Bool(true)})
	inputCreator.AssertOverride(t, components.Serverless, gqlschema.ConfigEntryInput{Key: "dockerRegistry.registryAddress", Value: "rtestinstanceid.azurecr.io", Secret: ptr.Bool(true)})

	// when retrying completed step
	op, when, err = step.Run(op, fixLogger())

	// then
	require.NoError(t, err)
	assert.Zero(t, when)
}

func Test_AzureContainerRegistryUnhappyPath(t *testing.T) {
	tests := []struct {
		name                string
		giveOperation       func(t *testing.T) internal.ProvisioningOperation
		giveStep            func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep
		wantRepeatOperation bool
	}{
		{
			name:          "Provision parameter errors",
			giveOperation: fixInvalidProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProvider()
				return *fixAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: false,
		},
		{
			name:          "AccountProvider cannot get gardener credentials",
			giveOperation: fixOperationAzureContainerRegistrySimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProviderGardenerCredentialsError()
				return *fixAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while getting config from HAP",
			giveOperation: fixOperationAzureContainerRegistrySimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProviderError(), accountProvider, fixAzureContainerRegistryStepConfig(), context.Background())
			},
			wantRepeatOperation: false,
		},
		{
			name: "Error while retrieving Azure Resource Group name",
			giveOperation: func(t *testing.T) internal.ProvisioningOperation {
				op := fixOperationAzureContainerRegistrySimple(t)
				op.Azure.ResourceGroupName = ""
				return op
			},
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProvider()
				return *fixAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while creating Azure Container Registry",
			giveOperation: fixOperationAzureContainerRegistrySimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientCreateContainerRegistryError()), accountProvider, fixAzureContainerRegistryStepConfig(), context.Background())
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while listing Azure Container Registry credentials",
			giveOperation: fixOperationAzureContainerRegistrySimple,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureContainerRegistryStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureContainerRegistryStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeAzureClientListContainerRegistryCredentialsError()), accountProvider, fixAzureContainerRegistryStepConfig(), context.Background())
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
func fixAzureContainerRegistryStep(memoryStorageOp storage.Operations, hyperscalerProvider azure.HyperscalerProvider,
	accountProvider *hyperscalerautomock.AccountProvider) *ProvisionAzureContainerRegistryStep {
	return NewProvisionAzureContainerRegistryStep(memoryStorageOp, hyperscalerProvider, accountProvider, fixAzureContainerRegistryStepConfig(), context.Background())
}

func fixOperationAzureContainerRegistrySimple(t *testing.T) internal.ProvisioningOperation {
	return fixOperationAzureContainerRegistry(newInputCreator())
}

func fixOperationAzureContainerRegistry(inputCreator *simpleInputCreator) internal.ProvisioningOperation {
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
		Azure: internal.AzureLifecycleData{
			ResourceGroupName: "kres-group",
		},
	}
	return op
}

func fixAzureContainerRegistryStepConfig() azure.StepConfig {
	return azure.StepConfig{
		ContainerRegistrySKU: containerregistry.Basic,
	}
}
