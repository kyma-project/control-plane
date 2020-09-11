package runtime

import (
	"time"

	"github.com/kyma-incubator/compass/components/director/pkg/pagination"
)

type runtimeDTO struct {
	InstanceID       string        `json:"instanceId"`
	RuntimeID        string        `json:"runtimeId"`
	GlobalAccountID  string        `json:"globalAccountId"`
	SubAccountID     string        `json:"subaccountId"`
	ShootName        string        `json:"shootName"`
	ServiceClassID   string        `json:"serviceClassID"`
	ServiceClassName string        `json:"serviceClassName"`
	ServicePlanID    string        `json:"servicePlanID"`
	ServicePlanName  string        `json:"servicePlanName"`
	Status           runtimeStatus `json:"status"`
}

type runtimeStatus struct {
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt"`
	Provisioning   *operation `json:"provisioning"`
	Deprovisioning *operation `json:"deprovisioning"`
	UpgradingKyma  *operation `json:"upgradingKyma"`
}

type operation struct {
	State       string `json:"state"`
	Description string `json:"description"`
}

type RuntimesPage struct {
	Data       []runtimeDTO     `json:"data"`
	PageInfo   *pagination.Page `json:"pageInfo"`
	TotalCount int              `json:"totalCount"`
}
