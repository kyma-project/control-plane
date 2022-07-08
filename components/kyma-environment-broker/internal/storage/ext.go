package storage

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

type Instances interface {
	FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error)
	FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error)
	FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error)
	GetByID(instanceID string) (*internal.Instance, error)
	Insert(instance internal.Instance) error
	Update(instance internal.Instance) (*internal.Instance, error)
	Delete(instanceID string) error
	GetInstanceStats() (internal.InstanceStats, error)
	GetERSContextStats() (internal.ERSContextStats, error)
	GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error)
	List(dbmodel.InstanceFilter) ([]internal.Instance, int, int, error)

	// todo: remove after instances parameters migration is done
	InsertWithoutEncryption(instance internal.Instance) error
	UpdateWithoutEncryption(instance internal.Instance) (*internal.Instance, error)
	ListWithoutDecryption(dbmodel.InstanceFilter) ([]internal.Instance, int, int, error)
}

type Operations interface {
	Provisioning
	Deprovisioning
	UpgradeKyma
	UpgradeCluster
	Updating

	GetLastOperation(instanceID string) (*internal.Operation, error)
	GetOperationByID(operationID string) (*internal.Operation, error)
	GetNotFinishedOperationsByType(operationType internal.OperationType) ([]internal.Operation, error)
	GetOperationStatsByPlan() (map[string]internal.OperationStats, error)
	GetOperationsForIDs(operationIDList []string) ([]internal.Operation, error)
	GetOperationStatsForOrchestration(orchestrationID string) (map[string]int, error)
	ListOperations(filter dbmodel.OperationFilter) ([]internal.Operation, int, int, error)
}

type Provisioning interface {
	InsertProvisioningOperation(operation internal.ProvisioningOperation) error
	GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error)
	GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error)
	UpdateProvisioningOperation(operation internal.ProvisioningOperation) (*internal.ProvisioningOperation, error)
	ListProvisioningOperationsByInstanceID(instanceID string) ([]internal.ProvisioningOperation, error)
}

type Deprovisioning interface {
	InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error
	GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error)
	GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error)
	UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error)
	ListDeprovisioningOperationsByInstanceID(instanceID string) ([]internal.DeprovisioningOperation, error)
	ListDeprovisioningOperations() ([]internal.DeprovisioningOperation, error)
}

type Orchestrations interface {
	Insert(orchestration internal.Orchestration) error
	Update(orchestration internal.Orchestration) error
	GetByID(orchestrationID string) (*internal.Orchestration, error)
	List(filter dbmodel.OrchestrationFilter) ([]internal.Orchestration, int, int, error)
}

type RuntimeStates interface {
	Insert(runtimeState internal.RuntimeState) error
	GetByOperationID(operationID string) (internal.RuntimeState, error)
	ListByRuntimeID(runtimeID string) ([]internal.RuntimeState, error)
	GetLatestByRuntimeID(runtimeID string) (internal.RuntimeState, error)
	GetLatestWithReconcilerInputByRuntimeID(runtimeID string) (internal.RuntimeState, error)
	GetLatestWithKymaVersionByRuntimeID(runtimeID string) (internal.RuntimeState, error)
	GetLatestWithOIDCConfigByRuntimeID(runtimeID string) (internal.RuntimeState, error)
}

type UpgradeKyma interface {
	InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error
	UpdateUpgradeKymaOperation(operation internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error)
	GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error)
	GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error)
	ListUpgradeKymaOperations() ([]internal.UpgradeKymaOperation, error)
	ListUpgradeKymaOperationsByInstanceID(instanceID string) ([]internal.UpgradeKymaOperation, error)
	ListUpgradeKymaOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeKymaOperation, int, int, error)
}

type UpgradeCluster interface {
	InsertUpgradeClusterOperation(operation internal.UpgradeClusterOperation) error
	UpdateUpgradeClusterOperation(operation internal.UpgradeClusterOperation) (*internal.UpgradeClusterOperation, error)
	GetUpgradeClusterOperationByID(operationID string) (*internal.UpgradeClusterOperation, error)
	ListUpgradeClusterOperationsByInstanceID(instanceID string) ([]internal.UpgradeClusterOperation, error)
	ListUpgradeClusterOperationsByOrchestrationID(orchestrationID string, filter dbmodel.OperationFilter) ([]internal.UpgradeClusterOperation, int, int, error)
}

type Updating interface {
	InsertUpdatingOperation(operation internal.UpdatingOperation) error
	GetUpdatingOperationByID(operationID string) (*internal.UpdatingOperation, error)
	ListUpdatingOperationsByInstanceID(instanceID string) ([]internal.UpdatingOperation, error)
	UpdateUpdatingOperation(operation internal.UpdatingOperation) (*internal.UpdatingOperation, error)
}
