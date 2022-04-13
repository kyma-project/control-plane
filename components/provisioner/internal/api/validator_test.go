package api

import (
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateProvisioningInput(t *testing.T) {
	clusterConfig, runtimeInput, kymaConfig := initializeConfigs()

	t.Run("Should return nil when config is correct", func(t *testing.T) {
		//given
		validator := NewValidator()

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

	t.Run("Should return nil when kyma config input not provided", func(t *testing.T) {
		//given
		validator := NewValidator()

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: clusterConfig,
		}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when config is incorrect", func(t *testing.T) {
		//given
		validator := NewValidator()

		config := gqlschema.ProvisionRuntimeInput{}

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when Runtime Agent component is not passed in installation config", func(t *testing.T) {
		//given
		validator := NewValidator()

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
		validator := NewValidator()

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

	t.Run("should return error when diskType or VolumeSizeGb is passed to openstack provisioning mutation", func(t *testing.T) {
		openStackClusterConfig := &gqlschema.ClusterConfigInput{
			GardenerConfig: &gqlschema.GardenerConfigInput{
				Name:                   "tets-clst",
				KubernetesVersion:      "1.15.4",
				VolumeSizeGb:           nil,
				MachineType:            "n1-standard-4",
				Region:                 "europe",
				Provider:               "openstack",
				Seed:                   util.StringPtr("2"),
				TargetSecret:           "test-secret",
				DiskType:               util.StringPtr("ssd"),
				WorkerCidr:             "10.10.10.10/255",
				AutoScalerMin:          1,
				AutoScalerMax:          3,
				MaxSurge:               40,
				MaxUnavailable:         1,
				ProviderSpecificConfig: nil,
			},
		}

		config := gqlschema.ProvisionRuntimeInput{
			RuntimeInput:  runtimeInput,
			ClusterConfig: openStackClusterConfig,
			KymaConfig:    kymaConfig,
		}

		validator := NewValidator()

		//when
		err := validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)

		openStackClusterConfig.GardenerConfig.VolumeSizeGb = util.IntPtr(30)
		openStackClusterConfig.GardenerConfig.DiskType = nil

		//when
		err = validator.ValidateProvisioningInput(config)

		//then
		require.Error(t, err)
	})
}

func TestValidator_ValidateUpgradeInput(t *testing.T) {

	t.Run("Should return nil when input is correct", func(t *testing.T) {
		//given
		validator := NewValidator()

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
		validator := NewValidator()

		config := gqlschema.UpgradeRuntimeInput{}

		//when
		err := validator.ValidateUpgradeInput(config)

		//then
		require.Error(t, err)
	})

	t.Run("Should return error when Runtime Agent component is not passed in kyma input", func(t *testing.T) {
		//given
		validator := NewValidator()

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
		validator := NewValidator()

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("1.20.8"),
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
		validator := NewValidator()

		config := gqlschema.UpgradeShootInput{}

		//when
		err := validator.ValidateUpgradeShootInput(config)

		//then
		require.Error(t, err)
		util.CheckErrorType(t, err, apperrors.CodeBadRequest)
	})

	t.Run("Should return error when Gardener config input provide empty value for machine type", func(t *testing.T) {
		//given
		validator := NewValidator()

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("1.20.8"),
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
		validator := NewValidator()

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("1.20.8"),
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
		validator := NewValidator()

		input := gqlschema.UpgradeShootInput{
			GardenerConfig: &gqlschema.GardenerUpgradeInput{
				KubernetesVersion:      util.StringPtr("1.20.8"),
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
		validator := NewValidator()

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

func initializeConfigs() (*gqlschema.ClusterConfigInput, *gqlschema.RuntimeInput, *gqlschema.KymaConfigInput) {
	clusterConfig := &gqlschema.ClusterConfigInput{
		GardenerConfig: &gqlschema.GardenerConfigInput{
			Name:                   "tets-clst",
			KubernetesVersion:      "1.15.4",
			VolumeSizeGb:           util.IntPtr(30),
			MachineType:            "n1-standard-4",
			Region:                 "europe",
			Provider:               "gcp",
			Seed:                   util.StringPtr("2"),
			TargetSecret:           "test-secret",
			DiskType:               util.StringPtr("ssd"),
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
