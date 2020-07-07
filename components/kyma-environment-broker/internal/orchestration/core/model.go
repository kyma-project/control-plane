package core

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Runtime is the data type which captures the needed SKR specific attributes to perform reconciliations on a given runtime.
type Runtime struct {
	internal.Instance

	ShootName              string
	MaintenanceWindowBegin string
	MaintenanceWindowEnd   string
}

const TargetAll = "all"

// RuntimeTarget captures a specification of SKR targets to resolve for an orchestration.
// When a RuntimeTarget defines multiple fields, all should match to any given runtime to be selected (i.e. the terms are AND-ed).
// Options:
//   - Target: all                                      // All SKRs provisioned successfully and not deprovisioning
//   - GlobalAccount: CA50125541TID000000000741207136   // regex pattern to match against, could also be CA.*
//   - SubAccount: 0d20e315-d0b4-48a2-9512-49bc8eb03cd1 // regex pattern to match against
//   - Region: "europe|eu-"                             // regex pattern to match against
type RuntimeTarget struct {
	Target        string
	GlobalAccount string
	SubAccount    string
	Region        string
}

// RuntimeResolver interface includes the Resolve() method, which, given an input slice of target specs to include and exclude,
// returns back a list of unique Runtime objects. The Runtime object contains attributes necessary for the executors and operations
// to perform reconcile on the given SKR. Such attributes include:
//   - Identifiers, like instance ID, global account, subaccount, (gardener shoot name)
//   - Shoot maintenance window
//
type RuntimeResolver interface {
	Resolve(include []RuntimeTarget, exclude []RuntimeTarget) ([]Runtime, error)
}

// RuntimeReconciler encapsulates a Runtime, and the logic to perform checking the actual vs. desired state
// of the specific operation use-case (e.g. Kyma release, shoot config) on the given runtime,
// as well as the logic to reconcile the SKR according to the specific use-case.
// The interface includes:
//   - Reconcile() gets the actual state, compares with desired state, and performs the operation on the specific SKR
//     when the actual state needs to be converged with the desired state.
//   - Runtime() returns the assigned runtime for this RuntimeReconciler
type RuntimeReconciler interface {
	Reconcile() error
	Runtime() Runtime
}

// NewRuntimeReconciler is the constructor" function type of any RuntimeReconciler implementation.
// The Runtime parameter holds the SKR specific attributes,
// while the ManagedOrchestration holds the desired state as well as methods for interacting with the specific custom resource object.
type NewRuntimeReconciler = func(r Runtime, o ManagedOrchestration) RuntimeReconciler

// ManagedOrchestration The various custom resources for orchestrating RuntimeReconcilers on SKRs implement this interface. It exposes the following:
//   - RuntimeTargetSpec() gets the runtime targets spec from the custom resource object
//   - OrchestrationSpec() gets the orchestration config spec from the custom resource object
//   - ReconcilerStatus() gets the current reconciler specific block of the custom resource status:
//   - UpdateReconcilerStatus is called by the OrchestrationReconciler implementation to update the reconciliation status inside the custom resource's status: attribute
type ManagedOrchestration interface {
	RuntimeTargetSpec() []RuntimeTarget
	OrchestrationSpec() OrchestrationSpec
	ReconcilerStatus() *ReconcilerStatus
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

// OrchestrationSpec is the spec part common for all ManagedOrchestration Custom Resources
type OrchestrationSpec struct {
	Strategy OrchestrationStrategyType
	Schedule OrchestrationScheduleType
	Parallel ParallelOrchestrationSpec
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

// OrchestrationStatus is the status part common for all ManagedOrchestration Custom Resources
type OrchestrationStatus struct {
	ObservedGeneration int
	Conditions         []OrchestrationCondition
	Reconciler         ReconcilerStatus
}
