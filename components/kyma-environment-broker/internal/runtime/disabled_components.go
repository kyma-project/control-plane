package runtime

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
)

// DisabledComponentsPerPlan provides a list of the components to remove per plan
// more specifically it's map[PLAN_ID or SELECTOR][COMPONENT_NAME]
//
// Components located under the AllPlansSelector will be removed from every plan
// All plans must be specified

func DisabledComponentsPerPlan() map[string]map[string]struct{} {
	return map[string]map[string]struct{}{
		broker.AllPlansSelector: {
			components.Backup:     {},
			components.BackupInit: {},
		},
		broker.GCPPlanID: {
			components.NatssStreaming:          {},
			components.KnativeProvisionerNatss: {},
		},
		broker.AzurePlanID: {
			components.NatssStreaming:          {},
			components.KnativeProvisionerNatss: {},
		},
		broker.AzureLitePlanID: {
			components.NatssStreaming:          {},
			components.KnativeProvisionerNatss: {},
		},
		broker.AzureTrialPlanID: {
			components.KnativeEventingKafka: {},
		},
		broker.GcpTrialPlanID: {
			components.KnativeEventingKafka: {},
		},
	}
}
