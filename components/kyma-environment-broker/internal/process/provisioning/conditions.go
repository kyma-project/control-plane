package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func WhenBTPOperatorCredentialsProvided(op internal.ProvisioningOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
