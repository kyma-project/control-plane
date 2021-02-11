package api

import (
	"testing"

	"github.com/kyma-incubator/hydroform/install/k8s"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	dbMocks "github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateProvisioningInput(t *testing.T) {

	t.Run("should validate correct on conflict values", func(t *testing.T) {
		// given
		validator, testConfig, config := onConflictValidatorInput()
		// when
		err := validator.ValidateProvisioningInput(config)
		// then
		require.NoError(t, err)

		// given
		for _, component := range testConfig.Components {
			component.OnConflict = util.StringPtr(k8s.ReplaceOnConflict)
		}
		// when
		err = validator.ValidateProvisioningInput(config)
		// then
		require.NoError(t, err)

		// given
		dummyValue := "dummy"
		testConfig.OnConflict = util.StringPtr(dummyValue)
		// when
		err = validator.ValidateProvisioningInput(config)
		// then
		require.Error(t, err)

		// given
		for _, component := range testConfig.Components {
			component.OnConflict = util.StringPtr(dummyValue)
		}
		// when
		err = validator.ValidateProvisioningInput(config)
		// then
		require.Error(t, err)
	})

	clusterConfig, runtimeInput, kymaConfig := initializeConfigs()

	t.Run("Should return nil when config is correct", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: clusterConfig,
			KymaConfig:    kymaConfig,
		}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when config is incorrect", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		config := gqlschema.ProvisionRuntimeInput{}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when Runtime Agent component is not passed in installation config", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
			},
		}

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: clusterConfig,
			KymaConfig:    kymaConfig,
		}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)
	})

	t.Run("should return error when machine image version is set, but machine image is empty", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		testClusterConfig := clusterConfig
		testClusterConfig.GardenerConfig.MachineImageVersion = util.StringPtr("24.3")

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: testClusterConfig,
			KymaConfig:    kymaConfig,
		}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)
	})
}

func TestValidator_ValidateUpgradeInput(t *testing.T) {

	t.Run("Should return nil when input is correct", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
				{
					Component:     "compass-runtime-agent",
					Configuration: nil,
				},
			},
		}

		input := gqlschema.UpgradeRuntimeInput{KymaConfig: kymaConfig}

		//when
		err := validator.ValidateUpgradeInput(input)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when kyma config input not provided", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		config := gqlschema.UpgradeRuntimeInput{}

		//when
		err := validator.ValidateUpgradeInput(config)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when Runtime Agent component is not passed in kyma input", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		kymaConfig := &gqlschema.KymaConfigInput{
			Version: "1.5",
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component:     "core",
					Configuration: nil,
				},
			},
		}

		input := gqlschema.UpgradeRuntimeInput{KymaConfig: kymaConfig}

		//when
		err := validator.ValidateUpgradeInput(input)

		//then
		require.Error(t, err)
	})
}

func TestValidator_ValidateUpgradeShootInput(t *testing.T) {

	t.Run("Should return nil when input is correct", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("version2"),
				MachineType:            util.StringPtr("new-machine"),
				DiskType:               util.StringPtr("papyrus"),
				Purpose:                util.StringPtr("development"),
				VolumeSizeGb:           util.IntPtr(50),
				AutoScalerMin:          util.IntPtr(2),
				AutoScalerMax:          util.IntPtr(6),
				MaxSurge:               util.IntPtr(2),
				MaxUnavailable:         util.IntPtr(1),
				ProviderSpecificConfig: nil,
			},
		}

		//when
		err := validator.ValidateUpgradeShootInput(input)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when Gardener config input not provided", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		config := gqlschema.UpgradeShootInput{}

		//when
		err := validator.ValidateUpgradeShootInput(config)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})

	t.Run("Should return error when Gardener config input provide empty value for machine type", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("version2"),
				MachineType:            util.StringPtr(""),
				DiskType:               util.StringPtr("stone"),
				Purpose:                util.StringPtr("development"),
				VolumeSizeGb:           util.IntPtr(50),
				AutoScalerMin:          util.IntPtr(2),
				AutoScalerMax:          util.IntPtr(6),
				MaxSurge:               util.IntPtr(2),
				MaxUnavailable:         util.IntPtr(1),
				ProviderSpecificConfig: nil,
			},
		}

		//when
		err := validator.ValidateUpgradeShootInput(input)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})

	t.Run("Should return error when Gardener config input provide empty value for disk type", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("version2"),
				MachineType:            util.StringPtr("time-machine"),
				DiskType:               util.StringPtr(""),
				Purpose:                util.StringPtr("evaluation"),
				VolumeSizeGb:           util.IntPtr(50),
				AutoScalerMin:          util.IntPtr(2),
				AutoScalerMax:          util.IntPtr(6),
				MaxSurge:               util.IntPtr(2),
				MaxUnavailable:         util.IntPtr(1),
				ProviderSpecificConfig: nil,
			},
		}

		//when
		err := validator.ValidateUpgradeShootInput(input)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})

	t.Run("Should return error when Gardener config input provide empty value for purpose", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("version2"),
				MachineType:            util.StringPtr("time-machine"),
				DiskType:               util.StringPtr("papyrus"),
				Purpose:                util.StringPtr(""),
				VolumeSizeGb:           util.IntPtr(50),
				AutoScalerMin:          util.IntPtr(2),
				AutoScalerMax:          util.IntPtr(6),
				MaxSurge:               util.IntPtr(2),
				MaxUnavailable:         util.IntPtr(1),
				ProviderSpecificConfig: nil,
			},
		}

		//when
		err := validator.ValidateUpgradeShootInput(input)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})

	t.Run("Should return error when Gardener config input provide empty value for kubernetes version", func(t *testing.T) {
		//given
		validator := NewValidator(nil)

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr(""),
				MachineType:            util.StringPtr("time-machine"),
				DiskType:               util.StringPtr("papyrus"),
				Purpose:                util.StringPtr("evaluation"),
				VolumeSizeGb:           util.IntPtr(50),
				AutoScalerMin:          util.IntPtr(2),
				AutoScalerMax:          util.IntPtr(6),
				MaxSurge:               util.IntPtr(2),
				MaxUnavailable:         util.IntPtr(1),
				ProviderSpecificConfig: nil,
			},
		}

		//when
		err := validator.ValidateUpgradeShootInput(input)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})
}

func TestValidator_ValidateTenant(t *testing.T) {
	tenant := "tenant"
	runtimeID := "123-123-123"
	t.Run("Should return nil when tenant matches tenant provided for Runtime", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		expectedTenant := "tenant"

		readSession.On("GetTenant", runtimeID).Return(expectedTenant, nil)

		//when
		err := validator.ValidateTenant(runtimeID, tenant)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when tenant does not match tenant provided for Runtime", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		expectedTenant := "otherTenant"

		readSession.On("GetTenant", runtimeID).Return(expectedTenant, nil)

		//when
		err := validator.ValidateTenant(runtimeID, tenant)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when persistence service returns error", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		readSession.On("GetTenant", runtimeID).Return("", dberrors.Internal("Some db error"))

		//when
		err := validator.ValidateTenant(runtimeID, tenant)

		//then
		require.Error(t, err)
	})
}

func TestValidator_ValidateTenantForOperation(t *testing.T) {
	tenant := "tenant"
	operationId := "123-123-123"

	t.Run("Should return nil when tenant matches tenant provided for Runtime", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		expectedTenant := "tenant"

		readSession.On("GetTenantForOperation", operationId).Return(expectedTenant, nil)

		//when
		err := validator.ValidateTenantForOperation(operationId, tenant)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when tenant does not match tenant provided for Runtime", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		expectedTenant := "otherTenant"

		readSession.On("GetTenantForOperation", operationId).Return(expectedTenant, nil)

		//when
		err := validator.ValidateTenantForOperation(operationId, tenant)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when persistence service returns error", func(t *testing.T) {
		//given
		readSession := &dbMocks.ReadSession{}
		validator := NewValidator(readSession)

		readSession.On("GetTenantForOperation", operationId).Return("", dberrors.Internal("Some db error"))

		//when
		err := validator.ValidateTenantForOperation(operationId, tenant)

		//then
		require.Error(t, err)
	})

}

func onConflictValidatorInput() (Validator, *gqlschema.KymaConfigInput, gqlschema.ProvisionRuntimeInput) {
	clusterConfig, runtimeInput, kymaConfig := initializeConfigs()
	readSession := &dbMocks.ReadSession{}
	validator := NewValidator(readSession)

	testConfig := kymaConfig
	testConfig.OnConflict = util.StringPtr(k8s.ReplaceOnConflict)

	config := gqlschema.ProvisionRuntimeInput{
		RuntimeInput:  runtimeInput,
		ClusterConfig: clusterConfig,
		KymaConfig:    testConfig,
	}
	return validator, testConfig, config
}

func initializeConfigs() (*gqlschema.ClusterConfigInput, *gqlschema.RuntimeInput, *gqlschema.KymaConfigInput) {
	clusterConfig := &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			KubernetesVersion:      "1.15.4",
			VolumeSizeGb:           30,
			MachineType:            "n1-standard-4",
			Region:                 "europe",
			Provider:               "gcp",
			Seed:                   util.StringPtr("2"),
			TargetSecret:           "test-secret",
			DiskType:               "ssd",
			WorkerCidr:             "10.10.10.10/255",
			AutoScalerMin:          1,
			AutoScalerMax:          3,
			MaxSurge:               40,
			MaxUnavailable:         1,
			ProviderSpecificConfig: nil,
		},
	}

	runtimeInput := &gqlschema.RuntimeInput{
		Name:        "test runtime",
		Description: new(string),
	}

	kymaConfig := &gqlschema.KymaConfigInput{
		Version: "1.5",
		Components: []*gqlschema.ComponentConfigurationInput{
			{
				Component:     "core",
				Configuration: nil,
			},
			{
				Component:     "compass-runtime-agent",
				Configuration: nil,
			},
		},
	}
	return clusterConfig, runtimeInput, kymaConfig
}
