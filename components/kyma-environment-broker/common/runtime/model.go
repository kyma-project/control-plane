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
	UserID           string        `json:"userID"`
}

type RuntimeStatus struct {
	CreatedAt      time.Time      `json:"createdAt"`
	ModifiedAt     time.Time      `json:"modifiedAt"`
	Provisioning   *Operation     `json:"provisioning"`
	Deprovisioning *Operation     `json:"deprovisioning,omitempty"`
	UpgradingKyma  OperationsData `json:"upgradingKyma,omitempty"`

	Suspension   OperationsData `json:"suspension,omitempty"`
	Unsuspension OperationsData `json:"unsuspension,omitempty"`
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
	OrchestrationID string    `json:"orchestrationID,omitempty"`
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
	PlanParam            = "plan"
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
	Plans            []string
}

type OperationType string

const (
	Provision    OperationType = "provision"
	Deprovision  OperationType = "deprovision"
	IpgradeKyma  OperationType = "kyma upgrade"
	Suspension   OperationType = "suspension"
	Unsuspension OperationType = "unsuspension"
)

func FindLastOperation(rt RuntimeDTO) (Operation, OperationType) {
	op := *rt.Status.Provisioning
	opType := Provision
	// Take the first upgrade operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.UpgradingKyma.Count > 0 {
		op = rt.Status.UpgradingKyma.Data[0]
		opType = IpgradeKyma
	}

	// Take the first unsuspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Unsuspension.Count > 0 && rt.Status.Unsuspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Unsuspension.Data[0]
		opType = Unsuspension
	}

	// Take the first suspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Suspension.Count > 0 && rt.Status.Suspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Suspension.Data[0]
		opType = Suspension
	}

	if rt.Status.Deprovisioning != nil && rt.Status.Deprovisioning.CreatedAt.After(op.CreatedAt) {
		op = *rt.Status.Deprovisioning
		opType = Deprovision
	}

	return op, opType
}
