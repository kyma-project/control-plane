package api

import (
	"github.com/kyma-incubator/hydroform/install/k8s"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const RuntimeAgent = "compass-runtime-agent"

//go:generate mockery -name=Validator
type Validator interface {
	ValidateProvisioningInput(input gqlschema.ProvisionRuntimeInput) apperrors.AppError
	ValidateUpgradeInput(input gqlschema.UpgradeRuntimeInput) apperrors.AppError
	ValidateUpgradeShootInput(input gqlschema.UpgradeShootInput) apperrors.AppError
	ValidateTenant(runtimeID, tenant string) apperrors.AppError
	ValidateTenantForOperation(operationID, tenant string) apperrors.AppError
}

type validator struct {
	readSession dbsession.ReadSession
}

func NewValidator(readSession dbsession.ReadSession) Validator {
	return &validator{
		readSession: readSession,
	}
}

func (v *validator) ValidateProvisioningInput(input gqlschema.ProvisionRuntimeInput) apperrors.AppError {
	if err := v.validateKymaConfig(input.KymaConfig); err != nil {
		return err.Append("Kyma config validation error while starting Runtime provisioning")
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
	err := v.validateKymaConfig(input.KymaConfig)
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

func (v *validator) ValidateTenant(runtimeID, tenant string) apperrors.AppError {
	dbTenant, err := v.readSession.GetTenant(runtimeID)
	if err != nil {
		return apperrors.Internal("Failed to get tenant from database: %s", err.Error())
	}

	if tenant != dbTenant {
		return apperrors.BadRequest("provided tenant does not match tenant used to provision cluster")
	}
	return nil
}

func (v *validator) ValidateTenantForOperation(operationID, tenant string) apperrors.AppError {
	dbTenant, err := v.readSession.GetTenantForOperation(operationID)
	if err != nil {
		return apperrors.Internal("Failed to get tenant from database: %s", err.Error())
	}

	if tenant != dbTenant {
		return apperrors.BadRequest("provided tenant does not match tenant used to provision cluster")
	}
	return nil
}

func (v *validator) validateKymaConfig(kymaConfig *gqlschema.KymaConfigInput) apperrors.AppError {
	if kymaConfig == nil {
		return apperrors.BadRequest("error: Kyma config not provided")
	}

	if appError, done := v.validateComponents(kymaConfig); done {
		return appError
	}

	if !configContainsRuntimeAgentComponent(kymaConfig.Components) {
		return apperrors.BadRequest("error: Kyma components list does not contain Compass Runtime Agent")
	}

	if kymaConfig.OnConflict != nil {
		return v.validateOnConflict(*kymaConfig.OnConflict)
	}

	return nil
}

func (v *validator) validateComponents(kymaConfig *gqlschema.KymaConfigInput) (apperrors.AppError, bool) {
	components := kymaConfig.Components
	if len(components) == 0 {
		return apperrors.BadRequest("error: Kyma components list is empty"), true
	}

	for _, component := range components {
		if component == nil || component.OnConflict == nil {
			continue
		}

		if err := v.validateOnConflict(*component.OnConflict); err != nil {
			return err, true
		}
	}

	return nil, false
}

func (v *validator) validateOnConflict(value string) apperrors.AppError {
	if value != "" && value != k8s.ReplaceOnConflict {
		return apperrors.BadRequest("error: Invalid value of conflict resolution onConflict")
	}

	return nil
}

func (v *validator) validateClusterConfig(clusterConfig *gqlschema.ClusterConfigInput) apperrors.AppError {
	if clusterConfig == nil || clusterConfig.GardenerConfig == nil {
		return apperrors.BadRequest("error: Cluster config with Gardener config not provided")
	}

	if err := v.validateMachineImage(*clusterConfig.GardenerConfig); err != nil {
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

func configContainsRuntimeAgentComponent(components []*gqlschema.ComponentConfigurationInput) bool {
	for _, component := range components {
		if component.Component == RuntimeAgent {
			return true
		}
	}
	return false
}
