package provisioning

import (
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	inputAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kymaVersion                   = "1.10"
	k8sVersion                    = "1.16.9"
	shootName                     = "c-1234567"
	instanceID                    = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID                   = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	globalAccountID               = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
	subAccountID                  = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	provisionerOperationID        = "1a0ed09b-9bb9-4e6f-a88c-01955c5f1129"
	runtimeID                     = "2498c8ee-803a-43c2-8194-6d6dd0354c30"
	autoUpdateKubernetesVersion   = true
	autoUpdateMachineImageVersion = true

	serviceManagerURL      = "http://sm.com"
	serviceManagerUser     = "admin"
	serviceManagerPassword = "admin123"
)

var shootPurpose = "evaluation"

func TestCheckClusterConfigurationStep_ClusterReady(t *testing.T) {
	st := storage.NewMemoryStorage()
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ClusterConfigurationVersion = 1
	recClient := reconciler.NewFakeClient()
	recClient.ApplyClusterConfig(reconcilerApi.Cluster{
		RuntimeID:    operation.RuntimeID,
		RuntimeInput: reconcilerApi.RuntimeInput{},
		KymaConfig:   reconcilerApi.KymaConfig{},
		Metadata:     reconcilerApi.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconcilerApi.StatusReady)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.NoError(t, err)
	assert.Zero(t, d)
}

func TestCheckClusterConfigurationStep_InProgress(t *testing.T) {
	for _, state := range []reconcilerApi.Status{reconcilerApi.StatusReconciling, reconcilerApi.StatusReconcilePending} {
		t.Run(fmt.Sprintf("shopuld repeat for state %s", state), func(t *testing.T) {
			st := storage.NewMemoryStorage()
			operation := fixture.FixProvisioningOperation("op-id", "inst-id")
			operation.ClusterConfigurationVersion = 1
			recClient := reconciler.NewFakeClient()
			recClient.ApplyClusterConfig(reconcilerApi.Cluster{
				RuntimeID:    operation.RuntimeID,
				RuntimeInput: reconcilerApi.RuntimeInput{},
				KymaConfig:   reconcilerApi.KymaConfig{},
				Metadata:     reconcilerApi.Metadata{},
				Kubeconfig:   "kubeconfig",
			})
			recClient.ChangeClusterState(operation.RuntimeID, 1, state)

			step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
			st.Operations().InsertProvisioningOperation(operation)

			// when
			_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

			// then
			require.NoError(t, err)
			assert.True(t, d > 0)
		})
	}
}

func TestCheckClusterConfigurationStep_ClusterFailed(t *testing.T) {
	st := storage.NewMemoryStorage()
	operation := fixture.FixProvisioningOperation("op-id", "inst-id")
	operation.ClusterConfigurationVersion = 1
	recClient := reconciler.NewFakeClient()
	recClient.ApplyClusterConfig(reconcilerApi.Cluster{
		RuntimeID:    operation.RuntimeID,
		RuntimeInput: reconcilerApi.RuntimeInput{},
		KymaConfig:   reconcilerApi.KymaConfig{},
		Metadata:     reconcilerApi.Metadata{},
		Kubeconfig:   "kubeconfig",
	})
	recClient.ChangeClusterState(operation.RuntimeID, 1, reconcilerApi.StatusError)

	step := NewCheckClusterConfigurationStep(st.Operations(), recClient, time.Minute)
	st.Operations().InsertProvisioningOperation(operation)

	// when
	_, d, err := step.Run(operation, logger.NewLogSpy().Logger)

	// then
	require.Error(t, err)
	assert.Zero(t, d)
}

func fixOperationCreateRuntime(t *testing.T, planID, region string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.InputCreator = fixInputCreator(t)
	provisioningOperation.InstanceDetails.ShootName = shootName
	provisioningOperation.InstanceDetails.ShootDNSProviders = gardener.DNSProvidersData{
		Providers: []gardener.DNSProviderData{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_intest",
				Type:           "route53_type_test",
			},
		},
	}
	provisioningOperation.ProvisioningParameters = FixProvisioningParameters(planID, region)
	provisioningOperation.RuntimeID = ""

	return provisioningOperation
}

func fixInstance() internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.GlobalAccountID = globalAccountID

	return instance
}

func FixProvisioningParameters(planID, region string) internal.ProvisioningParameters {
	return fixProvisioningParametersWithPlanID(planID, region)
}

func fixProvisioningParametersWithPlanID(planID, region string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:    planID,
		ServiceID: "",
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
			ServiceManager: &internal.ServiceManagerEntryDTO{
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: serviceManagerUser,
						Password: serviceManagerPassword,
					},
				},
				URL: serviceManagerURL,
			},
		},
		Parameters: internal.ProvisioningParametersDTO{
			Region: ptr.String(region),
			Name:   "dummy",
			Zones:  []string{"europe-west3-b", "europe-west3-c"},
		},
	}
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
	componentsProvider := &inputAutomock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := input.NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider, input.Config{
		KubernetesVersion:             k8sVersion,
		DefaultGardenerShootPurpose:   shootPurpose,
		AutoUpdateKubernetesVersion:   autoUpdateKubernetesVersion,
		AutoUpdateMachineImageVersion: autoUpdateMachineImageVersion,
	}, kymaVersion, fixTrialRegionMapping(), fixFreemiumProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID: broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{
			KymaVersion: "",
		},
	}
	version := internal.RuntimeVersionData{Version: kymaVersion, Origin: internal.Parameters}
	creator, err := ibf.CreateProvisionInput(pp, version)
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
