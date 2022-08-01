package dbmodel

import (
	"database/sql"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

// OperationFilter holds the filters when listing multiple operations
type OperationFilter struct {
	Page     int
	PageSize int
	States   []string
}

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
	FinishedStages         sql.NullString
	ProvisioningParameters sql.NullString

	Type internal.OperationType
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

type InstanceERSContextStatsEntry struct {
	LicenseType sql.NullString
	Total       int
}
