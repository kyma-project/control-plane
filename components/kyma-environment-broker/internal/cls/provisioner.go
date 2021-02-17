package cls

import (
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ProvisionerStorage --output=automock --outpkg=automock --case=underscore
type ProvisionerStorage interface {
	FindInstance(globalAccountID string) (*internal.CLSInstance, bool, error)
	InsertInstance(instance internal.CLSInstance) error
	Reference(version int, globalAccountID, skrInstanceID string) error
}

//go:generate mockery --name=InstanceCreator --output=automock --outpkg=automock --case=underscore
type InstanceCreator interface {
	CreateInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error
}

type provisioner struct {
	storage ProvisionerStorage
	creator InstanceCreator
	log     logrus.FieldLogger
}

func NewProvisioner(storage ProvisionerStorage, creator InstanceCreator, log logrus.FieldLogger) *provisioner {
	return &provisioner{
		storage: storage,
		creator: creator,
		log:     log,
	}
}

type ProvisionRequest struct {
	GlobalAccountID string
	Region          string
	SKRInstanceID   string
	ServiceID       string
	PlanID          string
	BrokerID        string
}

type ProvisionResult struct {
	InstanceID            string
	ProvisioningTriggered bool
}

func (c *provisioner) ProvisionIfNoneExists(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance, exists, err := c.storage.FindInstance(request.GlobalAccountID)
	if err != nil {
		return nil, errors.Wrapf(err, "while checking if instance is already created for global account: %s", request.GlobalAccountID)
	}

	if !exists {
		return c.createNewInstance(smClient, request)
	}

	err = c.storage.Reference(instance.Version, instance.GlobalAccountID, request.SKRInstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "while adding a reference to a cls instance with ID %s for global account: %s", instance.ID, request.GlobalAccountID)
	}

	return &ProvisionResult{
		InstanceID:            instance.ID,
		ProvisioningTriggered: false,
	}, nil
}

func (c *provisioner) createNewInstance(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance := internal.CLSInstance{
		ID:              uuid.New().String(),
		GlobalAccountID: request.GlobalAccountID,
		Region:          request.Region,
		CreatedAt:       time.Now(),
		SKRReferences:   []string{request.SKRInstanceID},
	}

	err := c.storage.InsertInstance(instance)
	if err != nil {
		return nil, errors.Wrapf(err, "while inserting a cls instance with ID %s for global account: %s", instance.ID, instance.GlobalAccountID)
	}

	err = c.creator.CreateInstance(smClient, servicemanager.InstanceKey{
		BrokerID:   request.BrokerID,
		ServiceID:  request.ServiceID,
		PlanID:     request.PlanID,
		InstanceID: instance.ID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "while creating instance name=%s", request.GlobalAccountID)
	}

	return &ProvisionResult{
		InstanceID:            instance.ID,
		ProvisioningTriggered: true,
	}, nil
}
