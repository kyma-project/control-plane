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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	fixKymaVersion                   = "1.19.0"
	fixKubernetesVersion             = "1.17.16"
	fixMachineType                   = "Standard_D8_v3"
	fixMachineImage                  = "gardenlinux"
	fixMachineImageVersion           = "184.0.0"
	fixAutoUpdateKubernetesVersion   = true
	fixAutoUpdateMachineImageVersion = true
)

func TestUpgradeKymaStep_Run(t *testing.T) {
	// given
	expectedOIDC := fixture.FixOIDCConfigDTO()
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixUpgradeClusterOperationWithInputCreator(t)
	err := memoryStorage.Operations().InsertUpgradeClusterOperation(operation)
	assert.NoError(t, err)

	provisioningOperation := fixProvisioningOperation()
	err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
	assert.NoError(t, err)

	runtimeState := fixture.FixRuntimeState("runtimestate-1", fixRuntimeID, provisioningOperation.ID)
	runtimeState.ClusterConfig.OidcConfig = &gqlschema.OIDCConfigInput{
		ClientID:       expectedOIDC.ClientID,
		GroupsClaim:    expectedOIDC.GroupsClaim,
		IssuerURL:      expectedOIDC.IssuerURL,
		SigningAlgs:    expectedOIDC.SigningAlgs,
		UsernameClaim:  expectedOIDC.UsernameClaim,
		UsernamePrefix: expectedOIDC.UsernamePrefix,
	}
	memoryStorage.RuntimeStates().Insert(runtimeState)
	assert.NoError(t, err)

	// as autoscaler values are not nil in provisioningParameters, the provider values are not used
	provider := fixGetHyperscalerProviderForPlanID(operation.ProvisioningParameters.PlanID)
	assert.NotNil(t, provider)

	provisionerClient := &provisionerAutomock.Client{}
	disabled := false
	provisionerClient.On("UpgradeShoot", fixGlobalAccountID, fixRuntimeID, gqlschema.UpgradeShootInput{
		GardenerConfig: &gqlschema.GardenerUpgradeInput{
			KubernetesVersion:                   ptr.String(fixKubernetesVersion),
			MachineType:                         ptr.String(fixMachineType),
			MachineImage:                        ptr.String(fixMachineImage),
			MachineImageVersion:                 ptr.String(fixMachineImageVersion),
			AutoScalerMin:                       operation.ProvisioningParameters.Parameters.AutoScalerMin,
			AutoScalerMax:                       operation.ProvisioningParameters.Parameters.AutoScalerMax,
			MaxSurge:                            operation.ProvisioningParameters.Parameters.MaxSurge,
			MaxUnavailable:                      operation.ProvisioningParameters.Parameters.MaxUnavailable,
			EnableKubernetesVersionAutoUpdate:   ptr.Bool(fixAutoUpdateKubernetesVersion),
			EnableMachineImageVersionAutoUpdate: ptr.Bool(fixAutoUpdateMachineImageVersion),
			ShootNetworkingFilterDisabled:       &disabled,
			OidcConfig: &gqlschema.OIDCConfigInput{
				ClientID:       expectedOIDC.ClientID,
				GroupsClaim:    expectedOIDC.GroupsClaim,
				IssuerURL:      expectedOIDC.IssuerURL,
				SigningAlgs:    expectedOIDC.SigningAlgs,
				UsernameClaim:  expectedOIDC.UsernameClaim,
				UsernamePrefix: expectedOIDC.UsernamePrefix,
			},
		},
		Administrators: []string{provisioningOperation.ProvisioningParameters.ErsContext.UserID},
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
	componentsProvider := &automock.ComponentListProvider{}
	defer componentsProvider.AssertExpectations(t)

	configProvider := &automock.ConfigurationProvider{}
	configProvider.On("ProvideForGivenVersionAndPlan",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).
		Return(&internal.ConfigForPlan{
			AdditionalComponents: []internal.KymaComponent{
				{
					Name:      "kyma-component1",
					Namespace: "kyma-system",
				},
			},
		}, nil)

	ibf, err := input.NewInputBuilderFactory(nil, nil, componentsProvider,
		configProvider, input.Config{
			KubernetesVersion:             fixKubernetesVersion,
			MachineImage:                  fixMachineImage,
			MachineImageVersion:           fixMachineImageVersion,
			TrialNodesNumber:              1,
			AutoUpdateKubernetesVersion:   fixAutoUpdateKubernetesVersion,
			AutoUpdateMachineImageVersion: fixAutoUpdateMachineImageVersion,
		}, fixKymaVersion, nil, nil, fixture.FixOIDCConfigDTO())
	require.NoError(t, err, "Input factory creation error")

	ver := internal.RuntimeVersionData{
		Version: fixKymaVersion,
		Origin:  internal.Defaults,
	}

	creator, err := ibf.CreateUpgradeShootInput(fixProvisioningParameters(), ver)
	require.NoError(t, err, "Input creator creation error")

	return creator
}
