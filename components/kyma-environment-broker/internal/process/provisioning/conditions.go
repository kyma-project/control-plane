package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func ForKyma2(op internal.ProvisioningOperation) bool {
	return op.RuntimeVersion.MajorVersion == 2
}

func ForKyma1(op internal.ProvisioningOperation) bool {
	return op.RuntimeVersion.MajorVersion == 1
}

func ForPlatformCredentialsProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.ServiceManager != nil
}

func ForBTPOperatorCredentialsProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
