package upgrade_cluster

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixKymaVersion         = "1.19.0"
	fixKubernetesVersion   = "1.17.16"
	fixMachineImage        = "gardenlinux"
	fixMachineImageVersion = "184.0.0"
)

func TestUpgradeKymaStep_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixUpgradeClusterOperationWithInputCreator(t)
	err := memoryStorage.Operations().InsertUpgradeClusterOperation(operation)
	assert.NoError(t, err)

	provisioningOperation := fixProvisioningOperation()
	err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
	assert.NoError(t, err)

	provider := fixGetHyperscalerProviderForPlanID(operation.ProvisioningParameters.PlanID)
	assert.NotNil(t, provider)
	//t.Logf("%v, %v, %v, %v", provider.Defaults().GardenerConfig.AutoScalerMin, provider.Defaults().GardenerConfig.AutoScalerMax, provider.Defaults().GardenerConfig.MaxSurge, provider.Defaults().GardenerConfig.MaxUnavailable)
	provisionerClient := &provisionerAutomock.Client{}
	provisionerClient.On("UpgradeShoot", fixGlobalAccountID, fixRuntimeID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:   ptr.String(fixKubernetesVersion),
			MachineImage:        ptr.String(fixMachineImage),
			MachineImageVersion: ptr.String(fixMachineImageVersion),
			AutoScalerMin:       ptr.Integer(provider.Defaults().GardenerConfig.AutoScalerMin),
			AutoScalerMax:       ptr.Integer(provider.Defaults().GardenerConfig.AutoScalerMax),
			MaxSurge:            ptr.Integer(provider.Defaults().GardenerConfig.MaxSurge),
			MaxUnavailable:      ptr.Integer(provider.Defaults().GardenerConfig.MaxUnavailable),
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       "",
				GroupsClaim:    "",
				IssuerURL:      "",
				SigningAlgs:    nil,
				UsernameClaim:  "",
				UsernamePrefix: "",
			},
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

	step := NewUpgradeClusterStep(memoryStorage.Operations(), memoryStorage.RuntimeStates(), provisionerClient, nil)

	// when

	operation, repeat, err := step.Run(operation, log.WithFields(logrus.Fields{"step": "TEST"}))

	// then
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, repeat)
	assert.Equal(t, fixProvisionerOperationID, operation.ProvisionerOperationID)
}

func fixUpgradeClusterOperationWithInputCreator(t *testing.T) internal.UpgradeClusterOperation {
	upgradeOperation := fixture.FixUpgradeClusterOperation(fixUpgradeOperationID, fixInstanceID)
	upgradeOperation.Description = ""
	upgradeOperation.ProvisioningParameters = fixProvisioningParameters()
	upgradeOperation.InstanceDetails.RuntimeID = fixRuntimeID
	upgradeOperation.RuntimeOperation.RuntimeID = fixRuntimeID
	upgradeOperation.RuntimeOperation.GlobalAccountID = fixGlobalAccountID
	upgradeOperation.RuntimeOperation.SubAccountID = fixSubAccountID
	upgradeOperation.InputCreator = fixInputCreator(t)

	return upgradeOperation
}

func fixInputCreator(t *testing.T) internal.ProvisionerInputCreator {
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
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", fixKymaVersion).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := input.NewInputBuilderFactory(nil, nil, componentsProvider, input.Config{
		KubernetesVersion:   fixKubernetesVersion,
		MachineImage:        fixMachineImage,
		MachineImageVersion: fixMachineImageVersion,
		TrialNodesNumber:    1,
	}, fixKymaVersion, nil, nil, fixture.FixOIDCConfigDTO())
	require.NoError(t, err, "Input factory creation error")

	creator, err := ibf.CreateUpgradeShootInput(fixProvisioningParameters())
	require.NoError(t, err, "Input creator creation error")

	return creator
}
