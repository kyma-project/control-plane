package dbmodel

import (
	"database/sql"
	"time"
)

type InstanceState string

const (
	InstanceSucceeded        InstanceState = "succeeded"
	InstanceFailed           InstanceState = "failed"
	InstanceProvisioning     InstanceState = "provisioning"
	InstanceDeprovisioning   InstanceState = "deprovisioning"
	InstanceUpgrading        InstanceState = "upgrading"
	InstanceDeprovisioned    InstanceState = "deprovisioned"
	InstanceNotDeprovisioned InstanceState = "notDeprovisioned"
)

// InstanceFilter holds the filters when querying Instances
type InstanceFilter struct {
	PageSize         int
	Page             int
	GlobalAccountIDs []string
	SubAccountIDs    []string
	InstanceIDs      []string
	RuntimeIDs       []string
	Regions          []string
	Plans            []string
	Domains          []string
	States           []InstanceState
}

type InstanceDTO struct {
	InstanceID      string
	RuntimeID       string
	GlobalAccountID string
	SubAccountID    string
	ServiceID       string
	ServiceName     string
	ServicePlanID   string
	ServicePlanName string

	DashboardURL           string
	ProvisioningParameters string
	ProviderRegion         string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time

	Version int
}

type InstanceWithOperationDTO struct {
	InstanceDTO

	Type        sql.NullString
	State       sql.NullString
	Description sql.NullString
}
