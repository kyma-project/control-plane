package runtime

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
)

type RuntimeDTO struct {
	InstanceID       string        `json:"instanceID"`
	RuntimeID        string        `json:"runtimeID"`
	GlobalAccountID  string        `json:"globalAccountID"`
	SubAccountID     string        `json:"subaccountID"`
	SubAccountRegion string        `json:"subaccountRegion"`
	ShootName        string        `json:"shootName"`
	ServiceClassID   string        `json:"serviceClassID"`
	ServiceClassName string        `json:"serviceClassName"`
	ServicePlanID    string        `json:"servicePlanID"`
	ServicePlanName  string        `json:"servicePlanName"`
	Status           runtimeStatus `json:"status"`
}

type runtimeStatus struct {
	CreatedAt      time.Time  `json:"createdAt"`
	ModifiedAt     time.Time  `json:"modifiedAt"`
	Provisioning   *operation `json:"provisioning"`
	Deprovisioning *operation `json:"deprovisioning,omitempty"`
	UpgradingKyma  *operation `json:"upgradingKyma,omitempty"`
}

type operation struct {
	State       string `json:"state"`
	Description string `json:"description"`
}

type RuntimesPage struct {
	Data       []RuntimeDTO     `json:"data"`
	PageInfo   *pagination.Page `json:"pageInfo"`
	TotalCount int              `json:"totalCount"`
}
