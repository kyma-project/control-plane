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

func WhenBTPOperatorCredentialsNotProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials == nil
}

func WhenBTPOperatorCredentialsProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
