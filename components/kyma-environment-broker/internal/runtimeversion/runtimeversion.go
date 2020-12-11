package runtimeversion

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type RuntimeVersionConfigurator struct {
	defaultVersion string
	accountMapping *AccountVersionMapping
}

func NewRuntimeVersionConfigurator(defaultVersion string, accountMapping *AccountVersionMapping) *RuntimeVersionConfigurator {
	return &RuntimeVersionConfigurator{
		defaultVersion: defaultVersion,
		accountMapping: accountMapping,
	}
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(op internal.ProvisioningOperation, pp internal.ProvisioningParameters) (*internal.RuntimeVersionData, error) {
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
	version, found, err := rvc.accountMapping.Get(op.GlobalAccountID, op.SubAccountID)
	if err != nil {
		return nil, err
	}
	if found {
		return internal.NewRuntimeVersionFromAccountMapping(version), nil
	}

	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}
