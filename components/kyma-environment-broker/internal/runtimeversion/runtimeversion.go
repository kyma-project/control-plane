package runtimeversion

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

type RuntimeVersionConfigurator struct {
	defaultVersion        string
	defaultPreviewVersion string
	accountMapping        *AccountVersionMapping
}

func NewRuntimeVersionConfigurator(defaultVersion string, previewVersion string, accountMapping *AccountVersionMapping) *RuntimeVersionConfigurator {
	return &RuntimeVersionConfigurator{
		defaultVersion:        defaultVersion,
		defaultPreviewVersion: previewVersion,
		accountMapping:        accountMapping,
	}
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(op internal.ProvisioningOperation) (*internal.RuntimeVersionData, error) {

	pp := op.ProvisioningParameters

	// TODO: Clarify: what about FromAccountMapping?
	if broker.IsPreviewPlan(pp.PlanID) && rvc.defaultPreviewVersion != "" {
		if pp.Parameters.KymaVersion != "" {
			return internal.NewRuntimeVersionFromParameters(pp.Parameters.KymaVersion), nil
		}
		return internal.NewRuntimeVersionFromDefaults(rvc.defaultPreviewVersion), nil
	}

	if pp.Parameters.KymaVersion == "" {
		version, found, err := rvc.accountMapping.Get(pp.ErsContext.GlobalAccountID, pp.ErsContext.SubAccountID)
		if err != nil {
			return nil, err
		}
		if found {
			return internal.NewRuntimeVersionFromAccountMapping(version), nil
		}
		return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
	}

	return internal.NewRuntimeVersionFromParameters(pp.Parameters.KymaVersion), nil
}

func (rvc *RuntimeVersionConfigurator) ForUpgrade(op internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	version, found, err := rvc.accountMapping.Get(op.GlobalAccountID, op.RuntimeOperation.SubAccountID)
	if err != nil {
		return nil, err
	}
	if found {
		return internal.NewRuntimeVersionFromAccountMapping(version), nil
	}

	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}
