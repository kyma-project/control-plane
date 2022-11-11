package update

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

func RequiresReconcilerUpdate(op internal.Operation) bool {
	// preview plan does not need any interaction with the Reconciler
	return op.RequiresReconcilerUpdate && !broker.IsPreviewPlan(op.ProvisioningParameters.PlanID)
}

func ForBTPOperatorCredentialsProvided(op internal.Operation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func CheckReconcilerStatus(op internal.Operation) bool {
	return op.CheckReconcilerStatus
}

func SkipForOwnClusterPlan(op internal.Operation) bool {
	return !broker.IsOwnClusterPlan(op.ProvisioningParameters.PlanID)
}
