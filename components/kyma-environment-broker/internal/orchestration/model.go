package orchestration

import (
	"time"
)

// Runtime is the data type which captures the needed SKR specific attributes to perform reconciliations on a given runtime.
type Runtime struct {
	InstanceID      string `json:"instanceId"`
	RuntimeID       string `json:"runtimeId"`
	GlobalAccountID string `json:"globalAccountId"`
	SubAccountID    string `json:"subaccountId"`
	// The corresponding shoot cluster's .metadata.name value
	ShootName string `json:"shootName"`
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.Begin value, in "HHMMSS+[TZ HHMM]" format, e.g. "040000+0000"
	MaintenanceWindowBegin string `json:"maintenanceWindowBegin"`
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.End value, in "HHMMSS+[TZ HHMM]" format, e.g. "040000+0000"
	MaintenanceWindowEnd string `json:"maintenanceWindowEnd"`
}

// RuntimeOperation encapsulates a Runtime object and an operation ID for the OrchestrationStrategy to execute.
type RuntimeOperation struct {
	Runtime
	OperationID string `json:"operationId"`
	Status      string `json:"status,omitempty"`
}

// TargetAll all SKRs provisioned successfully and not deprovisioning
const TargetAll = "all"

// RuntimeTarget captures a specification of SKR targets to resolve for an orchestration.
// When a RuntimeTarget defines multiple fields, all should match to any given runtime to be selected (i.e. the terms are AND-ed).
type RuntimeTarget struct {
	// Valid values: "all"
	Target string `json:"target,omitempty"`
	// Regex pattern to match against the runtime's GlobalAccount field. E.g. CA50125541TID000000000741207136, CA.*
	GlobalAccount string `json:"globalAccount,omitempty"`
	// Regex pattern to match against the runtime's SubAccount field. E.g. 0d20e315-d0b4-48a2-9512-49bc8eb03cd1
	SubAccount string `json:"subAccount,omitempty"`
	// Regex pattern to match against the shoot cluster's Region field (not SCP platform-region). E.g. "europe|eu-"
	Region string `json:"region,omitempty"`
}

// RuntimeResolver given an input slice of target specs to include and exclude, resolves and returns a list of unique Runtime objects.
type RuntimeResolver interface {
	Resolve(include []RuntimeTarget, exclude []RuntimeTarget) ([]Runtime, error)
}

// OrchestrationStrategy interface encapsulates the strategy how the orchestration is performed.
type OrchestrationStrategy interface {
	// Execute invokes operation managers' Execute(operationID string) method for each operation according to the encapsulated strategy.
	Execute(operations []RuntimeOperation) (time.Duration, error)
}

type OrchestrationStrategyType string

const (
	ParallelStrategy OrchestrationStrategyType = "parallel"
	CanaryStrategy   OrchestrationStrategyType = "canary"
)

type OrchestrationScheduleType string

const (
	Immediate         OrchestrationScheduleType = "immediate"
	MaintenanceWindow OrchestrationScheduleType = "maintenanceWindow"
)

// ParallelOrchestrationSpec
type ParallelOrchestrationSpec struct {
	Workers int `json:"workers"`
}

// OrchestrationStrategySpec is the strategy part common for all orchestration trigger/status API
type OrchestrationStrategySpec struct {
	Type     OrchestrationStrategyType `json:"type"`
	Schedule OrchestrationScheduleType `json:"schedule,omitempty"`
	Parallel ParallelOrchestrationSpec `json:"parallel,omitempty"`
}

// TargetSpec is the targets part common for all orchestration trigger/status API
type TargetSpec struct {
	Include []RuntimeTarget `json:"include"`
	Exclude []RuntimeTarget `json:"exclude,omitempty"`
}
