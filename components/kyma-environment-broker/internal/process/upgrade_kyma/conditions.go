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
	return op.ProvisioningParameters.ErsContext.ServiceManager != nil
}

func ForBTPOperatorCredentialsProvided(op internal.UpgradeKymaOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
