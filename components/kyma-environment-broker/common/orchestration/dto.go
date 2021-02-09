package orchestration

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

// Parameters hold the attributes of orchestration create (upgrade) requests.
type Parameters struct {
	Version  string       `json:"version"`
	Targets  TargetSpec   `json:"targets"`
	Strategy StrategySpec `json:"strategy,omitempty"`
	DryRun   bool         `json:"dryRun,omitempty"`
}

const (
	// StateParam parameter used in list orchestrations / operations queries to filter by state
	StateParam = "state"
)

// Orchestration states
const (
	Pending    = "pending"
	InProgress = "in progress"
	Canceling  = "canceling"
	Canceled   = "canceled"
	Succeeded  = "succeeded"
	Failed     = "failed"
)

// ListParameters hold attributes of list orchestrations / operations queries.
type ListParameters struct {
	Page     int
	PageSize int
	States   []string
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
	// RuntimeID is used to indicate a specific runtime
	RuntimeID string `json:"runtimeID,omitempty"`
	// PlanName is used to match runtimes with the same plan
	PlanName string `json:"planName,omitempty"`
	// Shoot is used to indicate a sepcific runtime by shoot name
	Shoot string `json:"shoot,omitempty"`
}

type StrategyType string

const (
	ParallelStrategy StrategyType = "parallel"
)

type ScheduleType string

const (
	Immediate         ScheduleType = "immediate"
	MaintenanceWindow ScheduleType = "maintenanceWindow"
)

// ParallelStrategySpec defines parameters for the parallel orchestration strategy
type ParallelStrategySpec struct {
	Workers int `json:"workers"`
}

// StrategySpec is the strategy part common for all orchestration trigger/status API
type StrategySpec struct {
	Type     StrategyType         `json:"type"`
	Schedule ScheduleType         `json:"schedule,omitempty"`
	Parallel ParallelStrategySpec `json:"parallel,omitempty"`
}

// TargetSpec is the targets part common for all orchestration trigger/status API
type TargetSpec struct {
	Include []RuntimeTarget `json:"include"`
	Exclude []RuntimeTarget `json:"exclude,omitempty"`
}

type StatusResponse struct {
	OrchestrationID string         `json:"orchestrationID"`
	State           string         `json:"state"`
	Description     string         `json:"description"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Parameters      Parameters     `json:"parameters"`
	OperationStats  map[string]int `json:"operationStats,omitempty"`
}

type OperationResponse struct {
	OperationID            string    `json:"operationID"`
	RuntimeID              string    `json:"runtimeID"`
	GlobalAccountID        string    `json:"globalAccountID"`
	SubAccountID           string    `json:"subAccountID"`
	OrchestrationID        string    `json:"orchestrationID"`
	ServicePlanID          string    `json:"servicePlanID"`
	ServicePlanName        string    `json:"servicePlanName"`
	DryRun                 bool      `json:"dryRun"`
	ShootName              string    `json:"shootName"`
	MaintenanceWindowBegin time.Time `json:"maintenanceWindowBegin"`
	MaintenanceWindowEnd   time.Time `json:"maintenanceWindowEnd"`
	State                  string    `json:"state"`
	Description            string    `json:"description"`
}

type OperationResponseList struct {
	Data       []OperationResponse `json:"data"`
	Count      int                 `json:"count"`
	TotalCount int                 `json:"totalCount"`
}

type OperationDetailResponse struct {
	OperationResponse

	KymaConfig    gqlschema.KymaConfigInput     `json:"kymaConfig"`
	ClusterConfig gqlschema.GardenerConfigInput `json:"clusterConfig"`
}

type StatusResponseList struct {
	Data       []StatusResponse `json:"data"`
	Count      int              `json:"count"`
	TotalCount int              `json:"totalCount"`
}

type UpgradeResponse struct {
	OrchestrationID string `json:"orchestrationID"`
}
