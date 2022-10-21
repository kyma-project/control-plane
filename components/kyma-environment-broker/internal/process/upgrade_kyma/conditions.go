package upgrade_kyma

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

func ForKyma2(op internal.UpgradeKymaOperation) bool {
	return op.RuntimeVersion.MajorVersion == 2
}

func ForKyma1(op internal.UpgradeKymaOperation) bool {
	return op.RuntimeVersion.MajorVersion == 1
}

func SkipForPreviewPlan(op internal.UpgradeKymaOperation) bool {
	return !broker.IsPreviewPlan(op.ProvisioningParameters.PlanID)
}

func WhenBTPOperatorCredentialsProvided(op internal.UpgradeKymaOperation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}
