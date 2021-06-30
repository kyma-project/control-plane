package runtime

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type State string

const (
	// StateSucceeded means that the last operation of the runtime has succeeded.
	StateSucceeded State = "succeeded"
	// StateFailed means that the last operation is one of provision, deprovivion, suspension, unsuspension, which has failed.
	StateFailed State = "failed"
	// StateError means the runtime is in a recoverable error state, due to the last upgrade operation has failed.
	StateError State = "error"
	// StateProvisioning means that the runtime provisioning (or unsuspension) is in progress (by the last runtime operation).
	StateProvisioning State = "provisioning"
	// StateDeprovisioning means that the runtime deprovisioning (or suspension) is in progress (by the last runtime operation).
	StateDeprovisioning State = "deprovisioning"
	// StateUpgrading means that kyma upgrade or cluster upgrade operation is in progress.
	StateUpgrading State = "upgrading"
	// StateSuspended means that the trial runtime is suspended (i.e. deprovisioned).
	StateSuspended State = "suspended"
	// AllState is a virtual state only used as query parameter in ListParameters to indicate "include all runtimes, which are excluded by default without state filters".
	AllState State = "all"
)

type RuntimeDTO struct {
	InstanceID              string                         `json:"instanceID"`
	RuntimeID               string                         `json:"runtimeID"`
	GlobalAccountID         string                         `json:"globalAccountID"`
	SubAccountID            string                         `json:"subAccountID"`
	ProviderRegion          string                         `json:"region"`
	SubAccountRegion        string                         `json:"subAccountRegion"`
	ShootName               string                         `json:"shootName"`
	ServiceClassID          string                         `json:"serviceClassID"`
	ServiceClassName        string                         `json:"serviceClassName"`
	ServicePlanID           string                         `json:"servicePlanID"`
	ServicePlanName         string                         `json:"servicePlanName"`
	Provider                string                         `json:"provider"`
	Status                  RuntimeStatus                  `json:"status"`
	UserID                  string                         `json:"userID"`
	AVSInternalEvaluationID int64                          `json:"avsInternalEvaluationID"`
	KymaConfig              *gqlschema.KymaConfigInput     `json:"kymaConfig,omitempty"`
	ClusterConfig           *gqlschema.GardenerConfigInput `json:"clusterConfig,omitempty"`
}

type RuntimeStatus struct {
	CreatedAt        time.Time       `json:"createdAt"`
	ModifiedAt       time.Time       `json:"modifiedAt"`
	State            State           `json:"state"`
	Provisioning     *Operation      `json:"provisioning,omitempty"`
	Deprovisioning   *Operation      `json:"deprovisioning,omitempty"`
	UpgradingKyma    *OperationsData `json:"upgradingKyma,omitempty"`
	UpgradingCluster *OperationsData `json:"upgradingCluster,omitempty"`
	Suspension       *OperationsData `json:"suspension,omitempty"`
	Unsuspension     *OperationsData `json:"unsuspension,omitempty"`
}

type OperationType string

const (
	Provision      OperationType = "provision"
	Deprovision    OperationType = "deprovision"
	UpgradeKyma    OperationType = "kyma upgrade"
	UpgradeCluster OperationType = "cluster upgrade"
	Suspension     OperationType = "suspension"
	Unsuspension   OperationType = "unsuspension"
)

type OperationsData struct {
	Data       []Operation `json:"data"`
	TotalCount int         `json:"totalCount"`
	Count      int         `json:"count"`
}

type Operation struct {
	State           string        `json:"state"`
	Type            OperationType `json:"type,omitempty"`
	Description     string        `json:"description"`
	CreatedAt       time.Time     `json:"createdAt"`
	OperationID     string        `json:"operationID"`
	OrchestrationID string        `json:"orchestrationID,omitempty"`
}

type RuntimesPage struct {
	Data       []RuntimeDTO `json:"data"`
	Count      int          `json:"count"`
	TotalCount int          `json:"totalCount"`
}

const (
	GlobalAccountIDParam = "account"
	SubAccountIDParam    = "subaccount"
	InstanceIDParam      = "instance_id"
	RuntimeIDParam       = "runtime_id"
	RegionParam          = "region"
	ShootParam           = "shoot"
	PlanParam            = "plan"
	StateParam           = "state"
	OperationDetailParam = "op_detail"
	KymaConfigParam      = "kyma_config"
	ClusterConfigParam   = "cluster_config"
)

type OperationDetail string

const (
	LastOperation OperationDetail = "last"
	AllOperation  OperationDetail = "all"
)

type ListParameters struct {
	// Page specifies the offset for the runtime results in the total count of matching runtimes
	Page int
	// PageSize specifies the count of matching runtimes returned in a response
	PageSize int
	// OperationDetail specifies whether the server should respond with all operations, or only the last operation. If not set, the server by default sends all operations
	OperationDetail OperationDetail
	// KymaConfig specifies whether kyma configuration details should be included in the response for each runtime
	KymaConfig bool
	// ClusterConfig specifies whether Gardener cluster configuration details should be included in the response for each runtime
	ClusterConfig bool
	// GlobalAccountIDs parameter filters runtimes by specified global account IDs
	GlobalAccountIDs []string
	// SubAccountIDs parameter filters runtimes by specified subaccount IDs
	SubAccountIDs []string
	// InstanceIDs parameter filters runtimes by specified instance IDs
	InstanceIDs []string
	// RuntimeIDs parameter filters runtimes by specified instance IDs
	RuntimeIDs []string
	// Regions parameter filters runtimes by specified provider regions
	Regions []string
	// Shoots parameter filters runtimes by specified shoot cluster names
	Shoots []string
	// Plans parameter filters runtimes by specified service plans
	Plans []string
	// States parameter filters runtimes by specified runtime states. See type State for possible values
	States []State
}

func (rt RuntimeDTO) LastOperation() Operation {
	op := Operation{}

	if rt.Status.Provisioning != nil {
		op = *rt.Status.Provisioning
		op.Type = Provision
	}
	// Take the first cluster upgrade operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.UpgradingCluster != nil && rt.Status.UpgradingCluster.Count > 0 {
		op = rt.Status.UpgradingCluster.Data[0]
		op.Type = UpgradeCluster
	}
	// Take the first upgrade operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.UpgradingKyma != nil && rt.Status.UpgradingKyma.Count > 0 && rt.Status.UpgradingKyma.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.UpgradingKyma.Data[0]
		op.Type = UpgradeKyma
	}

	// Take the first unsuspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Unsuspension != nil && rt.Status.Unsuspension.Count > 0 && rt.Status.Unsuspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Unsuspension.Data[0]
		op.Type = Unsuspension
	}

	// Take the first suspension operation, assuming that Data is sorted by CreatedAt DESC.
	if rt.Status.Suspension != nil && rt.Status.Suspension.Count > 0 && rt.Status.Suspension.Data[0].CreatedAt.After(op.CreatedAt) {
		op = rt.Status.Suspension.Data[0]
		op.Type = Suspension
	}

	if rt.Status.Deprovisioning != nil && rt.Status.Deprovisioning.CreatedAt.After(op.CreatedAt) {
		op = *rt.Status.Deprovisioning
		op.Type = Deprovision
	}

	return op
}
