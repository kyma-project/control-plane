package orchestration

import (
	"time"
)

// Runtime is the data type which captures the needed runtime specific attributes to perform orchestrations on a given runtime.
type Runtime struct {
	InstanceID      string `json:"instanceId,omitempty"`
	RuntimeID       string `json:"runtimeId"`
	GlobalAccountID string `json:"globalAccountId"`
	SubAccountID    string `json:"subaccountId"`
	// The corresponding shoot cluster's .metadata.name value
	ShootName string `json:"shootName"`
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.Begin value, which is in in "HHMMSS+[HHMM TZ]" format, e.g. "040000+0000"
	MaintenanceWindowBegin time.Time `json:"maintenanceWindowBegin"`
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.End value, which is in "HHMMSS+[HHMM TZ]" format, e.g. "040000+0000"
	MaintenanceWindowEnd time.Time `json:"maintenanceWindowEnd"`
}

// RuntimeOperation holds information about operation performed on a runtime
type RuntimeOperation struct {
	Runtime `json:""`
	ID      string `json:"-"`
	DryRun  bool   `json:"dryRun"`
}

//go:generate mockery --name=RuntimeResolver --output=automock --outpkg=automock --case=underscore
// RuntimeResolver given an input slice of target specs to include and exclude, resolves and returns a list of unique Runtime objects.
type RuntimeResolver interface {
	Resolve(targets TargetSpec) ([]Runtime, error)
}

// OperationExecutor implements methods to perform the operation corresponding to a Runtime.
type OperationExecutor interface {
	Execute(operationID string) (time.Duration, error)
	Reschedule(operationID string, maintenanceWindowBegin, maintenanceWindowEnd time.Time) error
}

//go:generate mockery --name=Strategy --output=automock --outpkg=automock --case=underscore
// Strategy interface encapsulates the strategy how the orchestration is performed.
type Strategy interface {
	// Execute invokes OperationExecutor's Execute(operationID string) method for each operation according to the encapsulated strategy.
	// The strategy is executed asynchronously. Successful call to the function returns a unique identifier, which can be used in a subsequent call to Wait().
	Execute(operations []RuntimeOperation, strategySpec StrategySpec) (string, error)
	// Wait blocks and waits until the execution with the given ID is finished.
	Wait(executionID string)
	// Cancel shutdowns a given execution.
	Cancel(executionID string)
}
