package update

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	inputAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeShootStep_Run(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()
	os := memoryStorage.Operations()
	rs := memoryStorage.RuntimeStates()
	cli := provisioner.NewFakeClient()
	step := NewUpgradeShootStep(os, rs, cli)
	operation := fixture.FixUpdatingOperation("op-id", "inst-id")
	operation.RuntimeID = "runtime-id"
	operation.ProvisionerOperationID = ""
	operation.ProvisioningParameters.Parameters.OIDC = &internal.OIDCConfigDTO{
		ClientID:       "client-id",
		GroupsClaim:    "groups",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RSA256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
	operation.InputCreator = fixInputCreator(t)
	os.InsertUpdatingOperation(operation)

	// when
	newOperation, d, err := step.Run(operation, logrus.New())
	require.NoError(t, err)
	assert.Zero(t, d)
	assert.True(t, cli.IsShootUpgraded("runtime-id"))
	req, _ := cli.LastShootUpgrade("runtime-id")
	assert.Equal(t, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "client-id",
				GroupsClaim:    "groups",
				IssuerURL:      "https://issuer.url",
				SigningAlgs:    []string{"RSA256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			},
		},
		Administrators: nil,
	}, req)
	assert.NotEmpty(t, newOperation.ProvisionerOperationID)
}

func fixInputCreator(t *testing.T) internal.ProvisionerInputCreator {
	optComponentsSvc := &inputAutomock.OptionalComponentService{}

	optComponentsSvc.On("ComputeComponentsToDisable", []string{}).Return([]string{})
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

	kymaComponentList := []v1alpha1.KymaComponent{
		{
			Name:      "to-remove-component",
			Namespace: "kyma-system",
		},
		{
			Name:      "keb",
			Namespace: "kyma-system",
		},
	}
	componentsProvider := &inputAutomock.ComponentListProvider{}
	const kymaVersion = "1.20"
	componentsProvider.On("AllComponents", kymaVersion).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	const k8sVersion = "1.18"
	ibf, err := input.NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider, input.Config{
		KubernetesVersion:           k8sVersion,
		DefaultGardenerShootPurpose: "test",
	}, kymaVersion, fixTrialRegionMapping(), fixFreemiumProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID: broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{
			KymaVersion: "",
		},
	}
	creator, err := ibf.CreateUpgradeShootInput(pp)
	if err != nil {
		t.Errorf("cannot create input creator for %q plan", broker.GCPPlanID)
	}

	return creator
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}

func fixFreemiumProviders() []string {
	return []string{"azure", "aws"}
}
