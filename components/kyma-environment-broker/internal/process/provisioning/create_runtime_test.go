package provisioning

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	inputAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	kymaVersion            = "1.10"
	k8sVersion             = "1.16.9"
	shootName              = "c-1234567"
	instanceID             = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID            = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	globalAccountID        = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
	subAccountID           = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	provisionerOperationID = "1a0ed09b-9bb9-4e6f-a88c-01955c5f1129"
	runtimeID              = "2498c8ee-803a-43c2-8194-6d6dd0354c30"

	serviceManagerURL      = "http://sm.com"
	serviceManagerUser     = "admin"
	serviceManagerPassword = "admin123"
)

var (
	shootPurpose = "development"
)

func TestCreateRuntimeStep_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationCreateRuntime(t, broker.GCPPlanID, "europe-west4-a")
	operation.ShootDomain = "kyma.org"
	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	administrator := ""
	profile := gqlschema.KymaProfileProduction
	strategy := gqlschema.ConflictStrategyReplace
	provisionerInput := gqlschema.ProvisionRuntimeInput{
		RuntimeInput: &gqlschema.RuntimeInput{
			Name:        "dummy",
			Description: nil,
			Labels: &gqlschema.Labels{
				"broker_instance_id":   instanceID,
				"global_subaccount_id": subAccountID,
				"operator_grafanaUrl":  "https://grafana.kyma.org",
			},
		},
		ClusterConfig: &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:              shootName,
				KubernetesVersion: k8sVersion,
				DiskType:          ptr.String("pd-standard"),
				VolumeSizeGb:      ptr.Integer(30),
				MachineType:       "n1-standard-4",
				Region:            "europe-west4-a",
				Provider:          "gcp",
				Purpose:           &shootPurpose,
				LicenceType:       nil,
				WorkerCidr:        "10.250.0.0/19",
				AutoScalerMin:     3,
				AutoScalerMax:     4,
				MaxSurge:          4,
				MaxUnavailable:    1,
				TargetSecret:      "",
				ProviderSpecificConfig: &gqlschema.ProviderSpecificInput{
					GcpConfig: &gqlschema.GCPProviderConfigInput{
						Zones: []string{"europe-west4-b", "europe-west4-c"},
					},
				},
				Seed: nil,
			},
			Administrators: []*string{&administrator},
		},
		KymaConfig: &gqlschema.KymaConfigInput{
			Version: kymaVersion,
			Components: internal.ComponentConfigurationInputList{
				{
					Component:     "keb",
					Namespace:     "kyma-system",
					Configuration: nil,
				},
			},
			Configuration:    []*gqlschema.ConfigEntryInput{},
			Profile:          &profile,
			ConflictStrategy: &strategy,
		},
	}

	provisionerClient := &provisionerAutomock.Client{}
	provisionerClient.On("ProvisionRuntime", globalAccountID, subAccountID, mock.MatchedBy(
		func(input gqlschema.ProvisionRuntimeInput) bool {
			return reflect.DeepEqual(input.RuntimeInput.Labels, provisionerInput.RuntimeInput.Labels) &&
				reflect.DeepEqual(input.KymaConfig, provisionerInput.KymaConfig) &&
				reflect.DeepEqual(input.ClusterConfig, provisionerInput.ClusterConfig)
		},
	)).Return(gqlschema.OperationStatus{
		ID:        ptr.String(provisionerOperationID),
		Operation: "",
		State:     "",
		Message:   nil,
		RuntimeID: ptr.String(runtimeID),
	}, nil)

	step := NewCreateRuntimeStep(memoryStorage.Operations(), memoryStorage.RuntimeStates(), memoryStorage.Instances(), provisionerClient)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	operation, repeat, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, repeat)
	assert.Equal(t, provisionerOperationID, operation.ProvisionerOperationID)

	instance, err := memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
	assert.Equal(t, instance.RuntimeID, runtimeID)
}

func TestCreateRuntimeStep_RunWithBadRequestError(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationCreateRuntime(t, broker.AzurePlanID, "westeurope")
	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	provisionerClient := &provisionerAutomock.Client{}
	provisionerClient.On("ProvisionRuntime", globalAccountID, subAccountID, mock.Anything).Return(gqlschema.OperationStatus{}, fmt.Errorf("some permanent error"))

	step := NewCreateRuntimeStep(memoryStorage.Operations(), memoryStorage.RuntimeStates(), memoryStorage.Instances(), provisionerClient)

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	operation, _, err = step.Run(operation, entry)

	// then
	assert.Equal(t, domain.Failed, operation.State)

}

func fixOperationCreateRuntime(t *testing.T, planID, region string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.InputCreator = fixInputCreator(t)
	provisioningOperation.InstanceDetails.ShootName = shootName
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
			Zones:  []string{"europe-west4-b", "europe-west4-c"},
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
	componentsProvider.On("AllComponents", kymaVersion).Return(kymaComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := input.NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider, input.Config{
		KubernetesVersion:           k8sVersion,
		DefaultGardenerShootPurpose: shootPurpose,
	}, kymaVersion, fixTrialRegionMapping(), fixFreemiumProviders())
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
