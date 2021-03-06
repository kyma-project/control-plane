package provisioning

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	hyperscalerautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	azuretesting "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure/testing"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	inputAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func fixLogger() logrus.FieldLogger {
	return logrus.StandardLogger()
}

func Test_HappyPath(t *testing.T) {
	// given
	tags := fixTags()
	memoryStorage := storage.NewMemoryStorage()
	accountProvider := fixAccountProvider()
	namespaceClient := azuretesting.NewFakeNamespaceClientHappyPath()
	step := fixEventHubStep(memoryStorage.Operations(), azuretesting.NewFakeHyperscalerProvider(namespaceClient), accountProvider)
	op := fixProvisioningOperation(t, broker.AzurePlanID, "westeurope")
	// this is required to avoid storage retries (without this statement there will be an error => retry)
	err := memoryStorage.Operations().InsertProvisioningOperation(op)
	require.NoError(t, err)

	// when
	op.UpdatedAt = time.Now()
	op, when, err := step.Run(op, fixLogger())
	require.NoError(t, err)
	provisionRuntimeInput, err := op.InputCreator.CreateProvisionRuntimeInput()
	require.NoError(t, err)

	// then
	ensureOperationSuccessful(t, op, when, err)
	allOverridesFound := ensureOverrides(t, provisionRuntimeInput)
	assert.True(t, allOverridesFound[components.KnativeEventing], "overrides for %s were not found", components.KnativeEventing)
	assert.True(t, allOverridesFound[components.KnativeEventingKafka], "overrides for %s were not found", components.KnativeEventingKafka)
	assert.Equal(t, namespaceClient.Tags, tags)
}

func Test_StepsUnhappyPath(t *testing.T) {
	tests := []struct {
		name                string
		giveOperation       func(t *testing.T, planID, region string) internal.ProvisioningOperation
		giveStep            func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep
		wantRepeatOperation bool
	}{
		{
			name:          "AccountProvider cannot get gardener credentials",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProviderGardenerCredentialsError()
				return *fixEventHubStep(storage.Operations(), azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeNamespaceClientHappyPath()), accountProvider)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "EventHubs Namespace creation error",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureEventHubStep(storage.Operations(),
					// ups ... namespace cannot get created
					azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeNamespaceClientCreationError()),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while getting EventHubs Namespace credentials",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureEventHubStep(storage.Operations(),
					// ups ... namespace cannot get listed
					azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeNamespaceClientListError()),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "No error while getting EventHubs Namespace credentials, but PrimaryConnectionString in AccessKey is nil",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureEventHubStep(storage.Operations(),
					// ups ... PrimaryConnectionString is nil
					azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeNamespaceAccessKeysNil()),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: true,
		},
		{
			name:          "Error while getting config from HAP",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureEventHubStep(storage.Operations(),
					// ups ... client cannot be created
					azuretesting.NewFakeHyperscalerProviderError(),
					accountProvider,
					context.Background(),
				)
			},
			wantRepeatOperation: false,
		},
		{
			name:          "Error while creating Azure ResourceGroup",
			giveOperation: fixProvisioningOperation,
			giveStep: func(t *testing.T, storage storage.BrokerStorage) ProvisionAzureEventHubStep {
				accountProvider := fixAccountProvider()
				return *NewProvisionAzureEventHubStep(storage.Operations(),
					// ups ... resource group cannot be created
					azuretesting.NewFakeHyperscalerProvider(azuretesting.NewFakeNamespaceResourceGroupError()),
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
			op := tt.giveOperation(t, broker.AzurePlanID, "westeurope")
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

func Test_getAzureResourceName(t *testing.T) {
	tests := []struct {
		name             string
		givenName        string
		wantResourceName string
	}{
		{
			name:             "all lowercase and starts with digit",
			givenName:        "1a23238d-1b04-3a9c-c139-405b75796ceb",
			wantResourceName: "k1a23238d-1b04-3a9c-c139-405b75796ceb",
		},
		{
			name:             "all uppercase and starts with digit",
			givenName:        "1A23238D-1B04-3A9C-C139-405B75796CEB",
			wantResourceName: "k1a23238d-1b04-3a9c-c139-405b75796ceb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getAzureResourceName(test.givenName)
			assert.Equal(t, test.wantResourceName, got)
		})
	}
}

// operationManager.OperationFailed(...)
// manager.go: if processedOperation.State != domain.InProgress { return 0, nil } => repeat
// queue.go: if err == nil && when != 0 => repeat

func ensureOperationIsRepeated(t *testing.T, err error, when time.Duration) {
	t.Helper()
	assert.Nil(t, err)
	assert.True(t, when != 0)
}

func ensureOperationIsNotRepeated(t *testing.T, err error) {
	t.Helper()
	assert.NotNil(t, err)
}

// ensureOverrides ensures that the overrides for
// - the kafka channel controller
// - and the default knative channel
// are set
func ensureOverrides(t *testing.T, provisionRuntimeInput gqlschema.ProvisionRuntimeInput) map[string]bool {
	t.Helper()

	allOverridesFound := map[string]bool{
		components.KnativeEventing:      false,
		components.KnativeEventingKafka: false,
	}

	kymaConfig := provisionRuntimeInput.KymaConfig
	for _, component := range kymaConfig.Components {
		switch component.Component {
		case components.KnativeEventing:
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "knative-eventing.channel.default.apiVersion",
				Value:  "knativekafka.kyma-project.io/v1alpha1",
				Secret: nil,
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "knative-eventing.channel.default.kind",
				Value:  "KafkaChannel",
				Secret: nil,
			})
			allOverridesFound[components.KnativeEventing] = true
		case components.KnativeEventingKafka:
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.brokers.hostname",
				Value:  "name",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.brokers.port",
				Value:  "9093",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.namespace",
				Value:  "knative-eventing",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.password",
				Value:  "Endpoint=sb://name/;",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.username",
				Value:  "$ConnectionString",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "kafka.secretName",
				Value:  "knative-kafka",
				Secret: ptr.Bool(true),
			})
			assert.Contains(t, component.Configuration, &gqlschema.ConfigEntryInput{
				Key:    "environment.kafkaProvider",
				Value:  kafkaProvider,
				Secret: ptr.Bool(true),
			})
			allOverridesFound[components.KnativeEventingKafka] = true
		}
	}

	return allOverridesFound
}

func fixKnativeKafkaInputCreator(t *testing.T) internal.ProvisionerInputCreator {
	optComponentsSvc := &inputAutomock.OptionalComponentService{}
	componentConfigurationInputList := internal.ComponentConfigurationInputList{
		{
			Component:     "keb",
			Namespace:     "kyma-system",
			Configuration: nil,
		},
		{
			Component: components.KnativeEventing,
			Namespace: "knative-eventing",
		},
		{
			Component: components.KnativeEventingKafka,
			Namespace: "knative-eventing",
		},
	}
	// "KnativeEventingKafka"
	optComponentsSvc.On("ComputeComponentsToDisable", []string{}).Return([]string{})
	optComponentsSvc.On("ExecuteDisablers", mock.Anything).Return(componentConfigurationInputList, nil)

	kymaComponentList := []v1alpha1.KymaComponent{
		{
			Name:      "keb",
			Namespace: "kyma-system",
		},
		{
			Name:      components.KnativeEventing,
			Namespace: "knative-eventing",
		},
		{
			Name:      components.KnativeEventingKafka,
			Namespace: "knative-eventing",
		},
	}
	componentsProvider := &inputAutomock.ComponentListProvider{}
	componentsProvider.On("AllComponents", kymaVersion).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := input.NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider, input.Config{}, kymaVersion, fixTrialRegionMapping())
	assert.NoError(t, err)
	pp := internal.ProvisioningParameters{
		PlanID: broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{
			KymaVersion: "",
		},
	}

	creator, err := ibf.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: kymaVersion, Origin: internal.Defaults})
	if err != nil {
		t.Errorf("cannot create input creator for %q plan", broker.GCPPlanID)
	}

	return creator
}

func fixAccountProvider() *hyperscalerautomock.AccountProvider {
	accountProvider := hyperscalerautomock.AccountProvider{}
	accountProvider.On("GardenerCredentials", hyperscaler.Azure, mock.Anything).Return(hyperscaler.Credentials{
		HyperscalerType: hyperscaler.Azure,
		CredentialData: map[string][]byte{
			"subscriptionID": []byte("subscriptionID"),
			"clientID":       []byte("clientID"),
			"clientSecret":   []byte("clientSecret"),
			"tenantID":       []byte("tenantID"),
		},
	}, nil)
	return &accountProvider
}

func fixAccountProviderGardenerCredentialsError() *hyperscalerautomock.AccountProvider {
	accountProvider := hyperscalerautomock.AccountProvider{}
	accountProvider.On("GardenerCredentials", hyperscaler.Azure, mock.Anything).Return(hyperscaler.Credentials{
		HyperscalerType: hyperscaler.Azure,
		CredentialData:  map[string][]byte{},
	}, fmt.Errorf("ups ... gardener credentials could not be retrieved"))
	return &accountProvider
}

func fixEventHubStep(memoryStorageOp storage.Operations, hyperscalerProvider azure.HyperscalerProvider,
	accountProvider *hyperscalerautomock.AccountProvider) *ProvisionAzureEventHubStep {
	return NewProvisionAzureEventHubStep(memoryStorageOp, hyperscalerProvider, accountProvider, context.Background())
}

func fixProvisioningOperation(t *testing.T, planID, region string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	provisioningOperation.ProvisioningParameters = fixProvisioningParameters(planID, region)
	provisioningOperation.InputCreator = fixKnativeKafkaInputCreator(t)
	provisioningOperation.State = ""

	return provisioningOperation
}

func fixTags() azure.Tags {
	return azure.Tags{
		azure.TagSubAccountID: ptr.String(subAccountID),
		azure.TagOperationID:  ptr.String(operationID),
		azure.TagInstanceID:   ptr.String(instanceID),
	}
}

func ensureOperationSuccessful(t *testing.T, op internal.ProvisioningOperation, when time.Duration, err error) {
	t.Helper()
	assert.Equal(t, when, time.Duration(0))
	assert.Equal(t, op.Operation.State, domain.LastOperationState(""))
	assert.Nil(t, err)
}
