package upgrade_kyma

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	kymaVersion  = "1.10.0"
	k8sVersion   = "1.16.9"
	shootPurpose = "development"
)

func TestUpgradeKymaStep_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixUpgradeKymaOperationWithInputCreator(t)
	err := memoryStorage.Operations().InsertUpgradeKymaOperation(operation)
	assert.NoError(t, err)

	provisioningOperation := fixProvisioningOperation()
	err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
	assert.NoError(t, err)
	profile := gqlschema.KymaProfileProduction
	strategy := gqlschema.ConflictStrategyReplace
	provisionerClient := &provisionerAutomock.Client{}
	provisionerClient.On("UpgradeRuntime", fixGlobalAccountID, fixRuntimeID, gqlschema.UpgradeRuntimeInput{
		KymaConfig: &gqlschema.KymaConfigInput{
			Version:          kymaVersion,
			Profile:          &profile,
			ConflictStrategy: &strategy,
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "keb",
					Namespace:     "kyma-system",
					Configuration: nil,
				},
			},
			Configuration: []*gqlschema.ConfigEntryInput{},
		},
	}).Return(gqlschema.OperationStatus{
		ID:        StringPtr(fixProvisionerOperationID),
		Operation: "",
		State:     "",
		Message:   nil,
		RuntimeID: StringPtr(fixRuntimeID),
	}, nil)
	provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
		ID:        ptr.String(fixProvisionerOperationID),
		Operation: "",
		State:     "",
		Message:   nil,
		RuntimeID: ptr.String(fixRuntimeID),
	}, nil)

	step := NewUpgradeKymaStep(memoryStorage.Operations(), memoryStorage.RuntimeStates(), provisionerClient, nil)

	// when

	operation, repeat, err := step.Run(operation, log.WithFields(logrus.Fields{"step": "TEST"}))

	// then
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, repeat)
	assert.Equal(t, fixProvisionerOperationID, operation.ProvisionerOperationID)
}

func fixUpgradeKymaOperationWithInputCreator(t *testing.T) internal.UpgradeKymaOperation {
	upgradeOperation := fixture.FixUpgradeKymaOperation(fixUpgradeOperationID, fixInstanceID)
	upgradeOperation.Description = ""
	upgradeOperation.ProvisioningParameters = fixProvisioningParameters()
	upgradeOperation.InstanceDetails.RuntimeID = fixRuntimeID
	upgradeOperation.RuntimeOperation.RuntimeID = fixRuntimeID
	upgradeOperation.InputCreator = fixInputCreator(t)

	return upgradeOperation
}

func fixInputCreator(t *testing.T) internal.ProvisionerInputCreator {
	optComponentsSvc := &automock.OptionalComponentService{}

	optComponentsSvc.On("ComputeComponentsToDisable", []string{}).Return([]string{})
	optComponentsSvc.On("AddComponentToDisable", mock.AnythingOfType("string"), mock.AnythingOfType("*runtime.GenericComponentDisabler")).Return(optComponentsSvc)
	optComponentsSvc.On("ExecuteDisablers", internal.ComponentConfigurationInputList{
		{
			Component:     "to-remove-component",
			Namespace:     "kyma-system",
			Configuration: nil,
		},
		{
			Component:     "keb",
			Namespace:     "kyma-system",
			Configuration: nil,
		},
	}).Return(internal.ComponentConfigurationInputList{
		{
			Component:     "keb",
			Namespace:     "kyma-system",
			Configuration: nil,
		},
	}, nil)

	kymaComponentList := []runtime.KymaComponent{
		{
			Name:      "to-remove-component",
			Namespace: "kyma-system",
		},
		{
			Name:      "keb",
			Namespace: "kyma-system",
		},
	}
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := input.NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider, input.Config{
		KubernetesVersion:           k8sVersion,
		DefaultGardenerShootPurpose: shootPurpose,
	}, kymaVersion, fixTrialRegionMapping(), []string{"azure", "aws"}, fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID: broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{
			KymaVersion: "",
		},
	}
	version := internal.RuntimeVersionData{Version: kymaVersion, Origin: internal.Defaults}
	creator, err := ibf.CreateUpgradeInput(pp, version)
	if err != nil {
		t.Errorf("cannot create input creator for %q plan", broker.GCPPlanID)
	}

	return creator
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}
