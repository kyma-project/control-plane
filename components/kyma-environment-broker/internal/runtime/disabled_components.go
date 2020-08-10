package runtime

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

// DisabledComponentsPerPlan provides a functionality to remove components per plan
// more specifically it's map[PLAN_ID or SELECTOR][COMPONENT_NAME]
//
// Components located under the AllPlansSelector will be removed from any plan
//

func DisabledComponentsPerPlan() map[string]map[string]struct{}{
	return map[string]map[string]struct{}{
		broker.AllPlansSelector: {
			"backup":                    {},
			"backup-init":               {},
		},
		broker.GCPPlanID: {
			"nats-streaming":            {},
			"knative-provisioner-natss": {},
		},
		broker.AzurePlanID: {
			"nats-streaming":            {},
			"knative-provisioner-natss": {},
		},
	}
}



