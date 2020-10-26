package orchestration

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type StatusResponse struct {
	OrchestrationID string                           `json:"orchestrationID"`
	State           string                           `json:"state"`
	Description     string                           `json:"description"`
	CreatedAt       time.Time                        `json:"createdAt"`
	UpdatedAt       time.Time                        `json:"updatedAt"`
	Parameters      internal.OrchestrationParameters `json:"parameters"`
}

type OperationResponse struct {
	OperationID            string    `json:"operationID"`
	RuntimeID              string    `json:"runtimeID"`
	GlobalAccountID        string    `json:"globalAccountID"`
	SubAccountID           string    `json:"subAccountID"`
	OrchestrationID        string    `json:"orchestrationID"`
	ServicePlanID          string    `json:"servicePlanID"`
	ServicePlanName        string    `json:"servicePlanName"`
	DryRun                 bool      `json:"dryRun"`
	ShootName              string    `json:"shootName"`
	MaintenanceWindowBegin time.Time `json:"maintenanceWindowBegin"`
	MaintenanceWindowEnd   time.Time `json:"maintenanceWindowEnd"`
	State                  string    `json:"state"`
	Description            string    `json:"description"`
}

type OperationResponseList struct {
	Data       []OperationResponse `json:"data"`
	Count      int                 `json:"count"`
	TotalCount int                 `json:"totalCount"`
}

type OperationDetailResponse struct {
	OperationResponse

	KymaConfig    gqlschema.KymaConfigInput     `json:"kymaConfig"`
	ClusterConfig gqlschema.GardenerConfigInput `json:"clusterConfig"`
}

type StatusResponseList struct {
	Data       []StatusResponse `json:"data"`
	Count      int              `json:"count"`
	TotalCount int              `json:"totalCount"`
}

type UpgradeResponse struct {
	OrchestrationID string `json:"orchestrationID"`
}
