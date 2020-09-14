package storage

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
)

type Instances interface {
	FindAllJoinedWithOperations(prct ...predicate.Predicate) ([]internal.InstanceWithOperation, error)
	FindAllInstancesForRuntimes(runtimeIdList []string) ([]internal.Instance, error)
	FindAllInstancesForSubAccounts(subAccountslist []string) ([]internal.Instance, error)
	GetByID(instanceID string) (*internal.Instance, error)
	Insert(instance internal.Instance) error
	Update(instance internal.Instance) error
	Delete(instanceID string) error
	GetInstanceStats() (internal.InstanceStats, error)
	GetNumberOfInstancesForGlobalAccountID(globalAccountID string) (int, error)
}

type Operations interface {
	Provisioning
	Deprovisioning
	UpgradeKyma

	GetOperationByID(operationID string) (*internal.Operation, error)
	GetOperationsInProgressByType(operationType dbmodel.OperationType) ([]internal.Operation, error)
	GetOperationStats() (internal.OperationStats, error)
	GetOperationsForIDs(operationIDList []string) ([]internal.Operation, error)
}

type Provisioning interface {
	InsertProvisioningOperation(operation internal.ProvisioningOperation) error
	GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error)
	GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error)
	UpdateProvisioningOperation(operation internal.ProvisioningOperation) (*internal.ProvisioningOperation, error)
}

type Deprovisioning interface {
	InsertDeprovisioningOperation(operation internal.DeprovisioningOperation) error
	GetDeprovisioningOperationByID(operationID string) (*internal.DeprovisioningOperation, error)
	GetDeprovisioningOperationByInstanceID(instanceID string) (*internal.DeprovisioningOperation, error)
	UpdateDeprovisioningOperation(operation internal.DeprovisioningOperation) (*internal.DeprovisioningOperation, error)
}

type Orchestrations interface {
	Insert(orchestration internal.Orchestration) error
	Update(orchestration internal.Orchestration) error
	GetByID(orchestrationID string) (*internal.Orchestration, error)
	ListAll() ([]internal.Orchestration, error)
}

type UpgradeKyma interface {
	InsertUpgradeKymaOperation(operation internal.UpgradeKymaOperation) error
	GetUpgradeKymaOperationByID(operationID string) (*internal.UpgradeKymaOperation, error)
	GetUpgradeKymaOperationByInstanceID(instanceID string) (*internal.UpgradeKymaOperation, error)
	UpdateUpgradeKymaOperation(operation internal.UpgradeKymaOperation) (*internal.UpgradeKymaOperation, error)
}

type LMSTenants interface {
	FindTenantByName(name, region string) (internal.LMSTenant, bool, error)
	InsertTenant(tenant internal.LMSTenant) error
}
