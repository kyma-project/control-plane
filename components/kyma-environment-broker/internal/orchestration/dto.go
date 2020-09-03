package orchestration

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type Parameters struct {
	Targets  internal.TargetSpec   `json:"targets"`
	Strategy internal.StrategySpec `json:"strategy,omitempty"`
	DryRun   bool                  `json:"dry_run,omitempty"`
}

type StatusResponse struct {
	OrchestrationID   string                      `json:"orchestration_id"`
	State             string                      `json:"state"`
	Description       string                      `json:"description"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	Parameters        Parameters                  `json:"parameters"`
	RuntimeOperations []internal.RuntimeOperation `json:"runtime_operations"`
}

type StatusResponseList struct {
	Orchestrations []StatusResponse `json:"orchestrations"`
}

type UpgradeResponse struct {
	OrchestrationID string `json:"orchestration_id"`
}
