package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
)

func WhenBTPOperatorCredentialsProvided(op internal.Operation) bool {
	return op.ProvisioningParameters.ErsContext.SMOperatorCredentials != nil
}

func SkipForOwnClusterPlan(operation internal.Operation) bool {
	return !broker.IsOwnClusterPlan(operation.ProvisioningParameters.PlanID)
}

func DoForOwnClusterPlanOnly(operation internal.Operation) bool {
	return !SkipForOwnClusterPlan(operation)
}
