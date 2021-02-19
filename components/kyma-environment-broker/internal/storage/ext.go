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

	GetLastOperation(instanceID string) (*internal.Operation, error)
	GetOperationByID(operationID string) (*internal.Operation, error)
	GetNotFinishedOperationsByType(operationType dbmodel.OperationType) ([]internal.Operation, error)
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
	ListByState(state string) ([]internal.Orchestration, error)
}

type RuntimeStates interface {
	Insert(runtimeState internal.RuntimeState) error
	GetByOperationID(operationID string) (internal.RuntimeState, error)
	ListByRuntimeID(runtimeID string) ([]internal.RuntimeState, error)
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

type LMSTenants interface {
	FindTenantByName(name, region string) (internal.LMSTenant, bool, error)
	InsertTenant(tenant internal.LMSTenant) error
}

type CLSInstances interface {
	FindActiveByGlobalAccountID(name string) (*internal.CLSInstance, bool, error)
	FindByID(clsInstanceID string) (*internal.CLSInstance, bool, error)
	Insert(instance internal.CLSInstance) error
	Reference(version int, clsInstanceID, skrInstanceID string) error
	Unreference(version int, clsInstanceID, skrInstanceID string) error
	MarkAsBeingRemoved(version int, clsInstanceID, skrInstanceID string) error
	Remove(clsInstanceID string) error
}
