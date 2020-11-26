package runtimeversion

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type RuntimeVersionConfigurator struct {
	defaultVersion       string
	globalAccountMapping *GlobalAccountVersionMapping
}

func NewRuntimeVersionConfigurator(defaultVersion string, globalAccountMapping *GlobalAccountVersionMapping) *RuntimeVersionConfigurator {
	return &RuntimeVersionConfigurator{
		defaultVersion:       defaultVersion,
		globalAccountMapping: globalAccountMapping,
	}
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(op internal.ProvisioningOperation, pp internal.ProvisioningParameters) (*internal.RuntimeVersionData, error) {
	if pp.Parameters.KymaVersion == "" {
		version, found, err := rvc.globalAccountMapping.Get(pp.ErsContext.GlobalAccountID)
		if err != nil {
			return nil, err
		}
		if found {
			return internal.NewRuntimeVersionFromGlobalAccount(version), nil
		}

		return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
	}

	return internal.NewRuntimeVersionFromParameters(pp.Parameters.KymaVersion), nil
}

func (rvc *RuntimeVersionConfigurator) ForUpgrade(op internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	version, found, err := rvc.globalAccountMapping.Get(op.GlobalAccountID)
	if err != nil {
		return nil, err
	}
	if found {
		return internal.NewRuntimeVersionFromGlobalAccount(version), nil
	}

	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}
