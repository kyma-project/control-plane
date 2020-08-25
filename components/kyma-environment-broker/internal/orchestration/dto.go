package orchestration

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

type UpgradeOrchestrationDTO struct {
	Targets  internal.TargetSpec   `json:"targets"`
	Strategy internal.StrategySpec `json:"strategy,omitempty"`
}

type UpgradeOrchestrationResponseDTO struct {
	OrchestrationID string `json:"orchestration_id"`
}
