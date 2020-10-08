package runtime

import (
	"time"
)

type RuntimeDTO struct {
	InstanceID       string        `json:"instanceID"`
	RuntimeID        string        `json:"runtimeID"`
	GlobalAccountID  string        `json:"globalAccountID"`
	SubAccountID     string        `json:"subaccountID"`
	ProviderRegion   string        `json:"region"`
	SubAccountRegion string        `json:"subaccountRegion"`
	ShootName        string        `json:"shootName"`
	ServiceClassID   string        `json:"serviceClassID"`
	ServiceClassName string        `json:"serviceClassName"`
	ServicePlanID    string        `json:"servicePlanID"`
	ServicePlanName  string        `json:"servicePlanName"`
	Status           RuntimeStatus `json:"status"`
}

type RuntimeStatus struct {
	CreatedAt      time.Time  `json:"createdAt"`
	ModifiedAt     time.Time  `json:"modifiedAt"`
	Provisioning   *Operation `json:"provisioning"`
	Deprovisioning *Operation `json:"deprovisioning,omitempty"`
	UpgradingKyma  *Operation `json:"upgradingKyma,omitempty"`
}

type Operation struct {
	State       string `json:"state"`
	Description string `json:"description"`
}

type RuntimesPage struct {
	Data       []RuntimeDTO `json:"data"`
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
}

const (
	PageSizeParam        = "page_size"
	PageParam            = "page"
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
