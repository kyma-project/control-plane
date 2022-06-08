package update

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func RequiresReconcilerUpdate(op internal.UpdatingOperation) bool {
	return op.RequiresReconcilerUpdate
}

func RequiresReconcilerUpdateForMigration(op internal.UpdatingOperation) bool {
	return ForMigration(op) && op.RequiresReconcilerUpdate
}

func ForPlatformCredentialsProvided(op internal.UpdatingOperation) bool {
	return op.ProvisioningParameters.ErsContext.ServiceManager != nil
}

func ForBTPOperatorCredentialsProvided(op internal.UpdatingOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func ForMigration(op internal.UpdatingOperation) bool {
	return op.InstanceDetails.SCMigrationTriggered
}

func CheckReconcilerStatus(op internal.UpdatingOperation) bool {
	return op.CheckReconcilerStatus
}
