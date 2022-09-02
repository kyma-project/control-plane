package update

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func RequiresReconcilerUpdate(op internal.UpdatingOperation) bool {
	return op.RequiresReconcilerUpdate
}

func ForBTPOperatorCredentialsProvided(op internal.UpdatingOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func CheckReconcilerStatus(op internal.UpdatingOperation) bool {
	return op.CheckReconcilerStatus
}
