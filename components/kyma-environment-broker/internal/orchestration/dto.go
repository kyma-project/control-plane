package orchestration

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type Parameters struct {
	Targets  internal.TargetSpec   `json:"targets"`
	Strategy internal.StrategySpec `json:"strategy,omitempty"`
	Dry      bool                  `json:"dry,omitempty"`
}

type UpgradeResponse struct {
	OrchestrationID string `json:"orchestration_id"`
}
