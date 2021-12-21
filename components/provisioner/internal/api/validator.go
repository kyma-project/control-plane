package api

import (
	"strings"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const RuntimeAgent = "compass-runtime-agent"

//go:generate mockery -name=Validator
type Validator interface {
	ValidateProvisioningInput(input gqlschema.ProvisionRuntimeInput) apperrors.AppError
	ValidateUpgradeInput(input gqlschema.UpgradeRuntimeInput) apperrors.AppError
	ValidateUpgradeShootInput(input gqlschema.UpgradeShootInput) apperrors.AppError
}

type validator struct {
}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateProvisioningInput(input gqlschema.ProvisionRuntimeInput) apperrors.AppError {
	if input.KymaConfig != nil {
		if err := v.validateKymaConfig(input.KymaConfig); err != nil {
			return err.Append("Kyma config validation error while starting Runtime provisioning")
		}
	}

	if input.RuntimeInput == nil {
		return apperrors.BadRequest("runtime input validation error while starting Runtime provisioning: runtime input is missing")
	}

	if err := v.validateClusterConfig(input.ClusterConfig); err != nil {
		return err.Append("Cluster config validation error while starting Runtime provisioning")
	}

	return nil
}

func (v *validator) ValidateUpgradeInput(input gqlschema.UpgradeRuntimeInput) apperrors.AppError {
	err := v.validateKymaConfigForUpgrade(input.KymaConfig)
	if err != nil {
		return err.Append("validation error while starting Runtime upgrade")
	}

	return nil
}

func (v *validator) ValidateUpgradeShootInput(input gqlschema.UpgradeShootInput) apperrors.AppError {

	config := input.GardenerConfig

	if config == nil {
		return apperrors.BadRequest("validation error while starting starting Shoot Upgrade: Gardener Config is missing")
	}

	if config.MachineType != nil && *config.MachineType == "" {
		return apperrors.BadRequest("empty machine type provided")
	}

	if config.KubernetesVersion != nil && *config.KubernetesVersion == "" {
		return apperrors.BadRequest("empty kubernetes version provided")
	}

	if config.DiskType != nil && *config.DiskType == "" {
		return apperrors.BadRequest("empty disk type provided")
	}

	if config.Purpose != nil && *config.Purpose == "" {
		return apperrors.BadRequest("empty purpose provided")
	}

	return nil
}

func (v *validator) validateKymaConfig(kymaConfig *gqlschema.KymaConfigInput) apperrors.AppError {
	if appError, done := v.validateComponents(kymaConfig); done {
		return appError
	}

	if !configContainsRuntimeAgentComponent(kymaConfig.Components) {
		return apperrors.BadRequest("error: Kyma components list does not contain Compass Runtime Agent")
	}

	return nil
}

func (v *validator) validateKymaConfigForUpgrade(kymaConfig *gqlschema.KymaConfigInput) apperrors.AppError {
	if kymaConfig == nil {
		return apperrors.BadRequest("error: Kyma config not provided")
	}
	return v.validateKymaConfig(kymaConfig)
}

func (v *validator) validateComponents(kymaConfig *gqlschema.KymaConfigInput) (apperrors.AppError, bool) {
	components := kymaConfig.Components
	if len(components) == 0 {
		return apperrors.BadRequest("error: Kyma components list is empty"), true
	}

	return nil, false
}

func (v *validator) validateClusterConfig(clusterConfig *gqlschema.ClusterConfigInput) apperrors.AppError {
	if clusterConfig == nil || clusterConfig.GardenerConfig == nil {
		return apperrors.BadRequest("error: Cluster config with Gardener config not provided")
	}

	gardenerConfig := *clusterConfig.GardenerConfig

	if err := v.validateMachineImage(gardenerConfig); err != nil {
		return err
	}

	if err := v.validateOpenStackVolume(gardenerConfig.DiskType, gardenerConfig.VolumeSizeGb, gardenerConfig.Provider); err != nil {
		return err
	}

	return nil
}

func (v *validator) validateMachineImage(gardenerConfig gqlschema.GardenerConfigInput) apperrors.AppError {
	if util.NotNilOrEmpty(gardenerConfig.MachineImageVersion) && util.IsNilOrEmpty(gardenerConfig.MachineImage) {
		return apperrors.BadRequest("error: Machine Image Version passed while Machine Image is empty")
	}
	return nil
}

// OpenStack does not accept diskType or volumeSize
func (v *validator) validateOpenStackVolume(diskType *string, volumeSizeGb *int, provider string) apperrors.AppError {
	if strings.ToLower(provider) == "openstack" {
		if diskType != nil || volumeSizeGb != nil {
			return apperrors.BadRequest("error: OpenStack mutation does not accept diskType or volumeSizeGb parameters")
		}
	}
	return nil
}

func configContainsRuntimeAgentComponent(components []*gqlschema.ComponentConfigurationInput) bool {
	for _, component := range components {
		if component.Component == RuntimeAgent {
			return true
		}
	}
	return false
}
