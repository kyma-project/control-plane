package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func WhenBTPOperatorCredentialsNotProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials == nil
}

func WhenBTPOperatorCredentialsProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
