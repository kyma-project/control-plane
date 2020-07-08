package core

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Runtime is the data type which captures the needed SKR specific attributes to perform reconciliations on a given runtime.
type Runtime struct {
	// Valid values: InstanceID, RuntimeID, GlobalAccountID, SubAccountID
	internal.Instance

	// The corresponding shoot cluster's .metadata.name value
	ShootName string
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.Begin value, in "HHMMSS+[TZ HHMM]" format, e.g. "040000+0000"
	MaintenanceWindowBegin string
	// The corresponding shoot cluster's .spec.maintenance.timeWindow.End value, in "HHMMSS+[TZ HHMM]" format, e.g. "040000+0000"
	MaintenanceWindowEnd string
}

// TargetAll all SKRs provisioned successfully and not deprovisioning
const TargetAll = "all"

// RuntimeTarget captures a specification of SKR targets to resolve for an orchestration.
// When a RuntimeTarget defines multiple fields, all should match to any given runtime to be selected (i.e. the terms are AND-ed).
type RuntimeTarget struct {
	// Valid values: "all"
	Target string
	// Regex pattern to match against the runtime's GlobalAccount field. E.g. CA50125541TID000000000741207136, CA.*
	GlobalAccount string
	// Regex pattern to match against the runtime's SubAccount field. E.g. 0d20e315-d0b4-48a2-9512-49bc8eb03cd1
	SubAccount string
	// Regex pattern to match against the shoot cluster's Region field (not SCP platform-region). E.g. "europe|eu-"
	Region string
}

// RuntimeResolver interface includes the Resolve() method, which, given an input slice of target specs to include and exclude,
// returns back a list of unique Runtime objects. The Runtime object contains attributes necessary for the executors and operations
// to perform reconcile on the given SKR.
type RuntimeResolver interface {
	Resolve(include []RuntimeTarget, exclude []RuntimeTarget) ([]Runtime, error)
}

// RuntimeReconciler encapsulates a Runtime, and the logic to perform checking the actual vs. desired state
// of the specific operation use-case (e.g. Kyma release, shoot config) on the given runtime,
// as well as the logic to reconcile the SKR according to the specific use-case.
type RuntimeReconciler interface {
	// Reconcile gets the actual state, compares with desired state, and performs the operation on the specific SKR when the actual state needs to be converged with the desired state.
	Reconcile() error
	// Runtime returns the assigned runtime for this RuntimeReconciler
	Runtime() Runtime
}

// NewRuntimeReconciler is the constructor" function type of any RuntimeReconciler implementation.
// The Runtime parameter holds the SKR specific attributes,
// while the ManagedOrchestration holds the desired state as well as methods for interacting with the specific custom resource object.
type NewRuntimeReconciler = func(r Runtime, o ManagedOrchestration) RuntimeReconciler

// ManagedOrchestration The various custom resources for orchestrating RuntimeReconcilers on SKRs implement this interface.
type ManagedOrchestration interface {
	// RuntimeTargetSpec gets the runtime targets spec from the custom resource object
	RuntimeTargetSpec() []RuntimeTarget
	// OrchestrationSpec gets the orchestration config spec from the custom resource object
	OrchestrationSpec() OrchestrationSpec
	// ReconcilerStatus gets the current reconciler specific block of the custom resource status:
	ReconcilerStatus() *ReconcilerStatus
	// UpdateReconcilerStatus is called by the OrchestrationReconciler implementation to update the reconciliation status inside the custom resource's status: attribute
	UpdateReconcilerStatus() error
}

// OrchestrationReconciler encapsulates the strategy how the orchestration is performed.
type OrchestrationReconciler interface {
	Reconcile() (reconcile.Result, error)
}

// NewOrchestrationReconciler the "constructor" function type of any OrchestrationReconciler implementation
type NewOrchestrationReconciler = func(orchestration ManagedOrchestration, resolver RuntimeResolver, newRuntimeReconciler NewRuntimeReconciler) OrchestrationReconciler

type OrchestrationStrategyType string

const (
	ParallelStrategy OrchestrationStrategyType = "Parallel"
	CanaryStrategy   OrchestrationStrategyType = "Canary"
)

type OrchestrationScheduleType string

const (
	Immediate         OrchestrationScheduleType = "Immediate"
	MaintenanceWindow OrchestrationScheduleType = "MaintenanceWindow"
)

// ParallelOrchestrationSpec
type ParallelOrchestrationSpec struct {
	Workers int
}

// OrchestrationSpec is the .spec.orchestration part common for all ManagedOrchestration Custom Resources
type OrchestrationSpec struct {
	Strategy OrchestrationStrategyType
	Schedule OrchestrationScheduleType
	Parallel ParallelOrchestrationSpec
}

// TargetSpec is the .spec.targets part common for all ManagedOrchestration Custom Resources
type TargetSpec struct {
	Include []RuntimeTarget
	Exclude []RuntimeTarget
}

// ReconcilerRuntimeStatistics
type ReconcilerRuntimeStatistics struct {
	Pending    int
	Succeeded  int
	Failed     int
	InProgress int
}

// ReconcilerRuntimeStatus
type ReconcilerRuntimeStatus struct {
	Stats ReconcilerRuntimeStatistics
}

// ReconcilerOperationStatus
type ReconcilerOperationStatus struct {
	State       string
	Description string
	Started     time.Time
	Updated     time.Time
}

// ReconcilerStatus
type ReconcilerStatus struct {
	LastOperation ReconcilerOperationStatus
	Runtimes      ReconcilerRuntimeStatus
}

// OrchestrationCondition
type OrchestrationCondition struct {
	Type           string
	Reason         string
	Status         string
	Message        string
	LastUpdateTime time.Time
}

// OrchestrationStatus is the .status part common for all ManagedOrchestration Custom Resources
type OrchestrationStatus struct {
	ObservedGeneration int
	Conditions         []OrchestrationCondition
	Reconciler         ReconcilerStatus
}
