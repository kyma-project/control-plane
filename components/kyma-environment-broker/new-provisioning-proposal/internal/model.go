package internal

import (
	"github.com/pivotal-cf/brokerapi/v7/domain"
)

type Operation struct {
	ID         string
	InstanceID string
	State      domain.LastOperationState
	Version    int
}

// ProvisioningOperation holds all information about provisioning operation
type ProvisioningOperation struct {
	Operation `json:"-"`

	ProvisioningParameters ProvisioningParameters
	LMS                    string
	EDP                    string
	Runtime                string
}

type ProvisioningParameters struct {
	PlanID string
}

func NewProvisioningOperationWithID(operationID, instanceID string, parameters ProvisioningParameters) (ProvisioningOperation, error) {
	return ProvisioningOperation{
		Operation: Operation{
			ID:         operationID,
			Version:    0,
			InstanceID: instanceID,
			State:      domain.InProgress,
		},
		ProvisioningParameters: parameters,
	}, nil
}
