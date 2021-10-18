package upgrade_kyma

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func ForKyma2(op internal.UpgradeKymaOperation) bool {
	return op.RuntimeVersion.MajorVersion == 2
}

func ForKyma1(op internal.UpgradeKymaOperation) bool {
	return op.RuntimeVersion.MajorVersion == 1
}

func ForPlatformCredentialsProvided(op internal.UpgradeKymaOperation) bool {
	if op.ProvisioningParameters.ErsContext.ServiceManager != nil {
		if op.ProvisioningParameters.ErsContext.ServiceManager.Credentials != nil {
			return true
		}
	}
	return false
}

func ForBTPOperatorCredentialsProvided(op internal.UpgradeKymaOperation) bool {
	if op.ProvisioningParameters.ErsContext.ServiceManager != nil {
		if op.ProvisioningParameters.ErsContext.ServiceManager.BTPOperatorCredentials != nil {
			return true
		}
	}
	return false
}
