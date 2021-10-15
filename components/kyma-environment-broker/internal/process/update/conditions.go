package update

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

func ForKyma2(op internal.UpdatingOperation) bool {
	return op.RuntimeVersion.MajorVersion == 2
}

func ForKyma1(op internal.UpdatingOperation) bool {
	return op.RuntimeVersion.MajorVersion == 1
}

func ForMigration(op internal.UpdatingOperation) bool {
	if op.ProvisioningParameters.ErsContext.ServiceManager == nil {
		return false
	}
	return op.ProvisioningParameters.ErsContext.ServiceManager.IsMigrationFromSCtoOperator || op.InstanceDetails.SCMigrationTriggered
}

func ForMigrationOrKyma2(op internal.UpdatingOperation) bool {
	return ForMigration(op) || ForKyma2(op)
}
