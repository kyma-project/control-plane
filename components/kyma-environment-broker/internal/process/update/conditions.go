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

func ForPlatformCredentialsProvided(op internal.UpdatingOperation) bool {
	return op.ProvisioningParameters.ErsContext.ServiceManager != nil
}

func ForBTPOperatorCredentialsProvided(op internal.UpdatingOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func ForMigration(op internal.UpdatingOperation) bool {
	return op.InstanceDetails.SCMigrationTriggered
}
