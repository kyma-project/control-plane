package dbmodel

import (
	"database/sql"
	"time"
)

// OperationFilter holds the filters when listing multiple operations
type OperationFilter struct {
	Page     int
	PageSize int
	States   []string
}

// OperationType defines the possible types of an asynchronous operation to a broker.
type OperationType string

const (
	// OperationTypeProvision means provisioning OperationType
	OperationTypeProvision OperationType = "provision"
	// OperationTypeDeprovision means deprovision OperationType
	OperationTypeDeprovision OperationType = "deprovision"
	// OperationTypeUndefined means undefined OperationType
	OperationTypeUndefined OperationType = ""
	// OperationTypeUpgradeKyma means upgrade Kyma OperationType
	OperationTypeUpgradeKyma OperationType = "upgradeKyma"
)

type OperationDTO struct {
	ID        string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time

	InstanceID        string
	OrchestrationID   sql.NullString
	TargetOperationID string

	Data                   string
	State                  string
	Description            string
	ProvisioningParameters sql.NullString

	Type OperationType
}

type OperationStatEntry struct {
	Type   string
	State  string
	PlanID string
}

type InstanceByGlobalAccountIDStatEntry struct {
	GlobalAccountID string
	Total           int
}
