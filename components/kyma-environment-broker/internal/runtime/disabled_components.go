package runtime

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
)

//
// DisabledComponentsProvider provides a list of the components to disable per a specific plan
// more specifically it's map[PLAN_ID or SELECTOR][COMPONENT_NAME]
//
// Components located under the AllPlansSelector will be removed from every plan
// All plans must be specified
//

type DisabledComponentsProvider map[string]map[string]struct{}

func NewDisabledComponentsProvider() DisabledComponentsProvider {
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

func (p DisabledComponentsProvider) DisabledComponentsPerPlan(planID string) (map[string]struct{}, error) {
	if _, ok := p[planID]; !ok {
		return nil, fmt.Errorf("unknown plan %s", planID)
	}
	return p[planID], nil
}

func (p DisabledComponentsProvider) DisabledForAll() map[string]struct{} {
	return p[broker.AllPlansSelector]
}
