package api

import (
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
		return err.Append("Cluster config validatin error while starting Runtime provisioning")
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

	if len(kymaConfig.Components) == 0 {
		return apperrors.BadRequest("error: Kyma components list is empty")
	}

	if !configContainsRuntimeAgentComponent(kymaConfig.Components) {
		return apperrors.BadRequest("error: Kyma components list does not contain Compass Runtime Agent")
	}

	return nil
}

func (v *validator) validateClusterConfig(clusterConfig *gqlschema.ClusterConfigInput) apperrors.AppError {
	if clusterConfig == nil || clusterConfig.GardenerConfig == nil {
		return apperrors.BadRequest("error: Cluster config with Gardener config not provided")
	}

	if util.NotNilOrEmpty(clusterConfig.GardenerConfig.MachineImageVersion) &&
		util.IsNilOrEmpty(clusterConfig.GardenerConfig.MachineImage) {

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
