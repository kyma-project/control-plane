package orchestration

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type StatusResponse struct {
	OrchestrationID   string                           `json:"orchestration_id"`
	State             string                           `json:"state"`
	Description       string                           `json:"description"`
	CreatedAt         time.Time                        `json:"created_at"`
	UpdatedAt         time.Time                        `json:"updated_at"`
	Parameters        internal.OrchestrationParameters `json:"parameters"`
	RuntimeOperations []internal.RuntimeOperation      `json:"runtime_operations,omitempty"`
}

type StatusResponseList struct {
	Data []StatusResponse `json:"data"`
}

type UpgradeResponse struct {
	OrchestrationID string `json:"orchestration_id"`
}
