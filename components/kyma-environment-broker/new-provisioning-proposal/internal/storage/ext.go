package storage

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal"

type Operations interface {
	Provisioning

	GetOperationByID(operationID string) (*internal.Operation, error)
}

type Provisioning interface {
	InsertProvisioningOperation(operation internal.ProvisioningOperation) error
	GetProvisioningOperationByID(operationID string) (*internal.ProvisioningOperation, error)
	GetProvisioningOperationByInstanceID(instanceID string) (*internal.ProvisioningOperation, error)
	UpdateProvisioningOperation(operation internal.ProvisioningOperation) (*internal.ProvisioningOperation, error)
}
