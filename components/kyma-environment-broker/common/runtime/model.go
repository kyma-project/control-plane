package runtime

import (
	"time"
)

type RuntimeDTO struct {
	InstanceID       string        `json:"instanceID"`
	RuntimeID        string        `json:"runtimeID"`
	GlobalAccountID  string        `json:"globalAccountID"`
	SubAccountID     string        `json:"subAccountID"`
	ProviderRegion   string        `json:"region"`
	SubAccountRegion string        `json:"subAccountRegion"`
	ShootName        string        `json:"shootName"`
	ServiceClassID   string        `json:"serviceClassID"`
	ServiceClassName string        `json:"serviceClassName"`
	ServicePlanID    string        `json:"servicePlanID"`
	ServicePlanName  string        `json:"servicePlanName"`
	Status           RuntimeStatus `json:"status"`
}

type RuntimeStatus struct {
	CreatedAt      time.Time      `json:"createdAt"`
	ModifiedAt     time.Time      `json:"modifiedAt"`
	Provisioning   *Operation     `json:"provisioning"`
	Deprovisioning *Operation     `json:"deprovisioning,omitempty"`
	UpgradingKyma  OperationsData `json:"upgradingKyma,omitempty"`
}

type OperationsData struct {
	Data       []Operation `json:"data"`
	TotalCount int         `json:"totalCount"`
	Count      int         `json:"count"`
}

type Operation struct {
	State           string    `json:"state"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"createdAt"`
	OperationID     string    `json:"operationID"`
	OrchestrationID *string   `json:"orchestrationID,omitempty"`
}

type RuntimesPage struct {
	Data       []RuntimeDTO `json:"data"`
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
}

const (
	GlobalAccountIDParam = "account"
	SubAccountIDParam    = "subaccount"
	InstanceIDParam      = "instance_id"
	RuntimeIDParam       = "runtime_id"
	RegionParam          = "region"
	ShootParam           = "shoot"
)

type ListParameters struct {
	Page             int
	PageSize         int
	GlobalAccountIDs []string
	SubAccountIDs    []string
	InstanceIDs      []string
	RuntimeIDs       []string
	Regions          []string
	Shoots           []string
}
