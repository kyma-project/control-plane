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
	Region                string
}

func (p *provisioner) Provision(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance, exists, err := p.storage.FindInstance(request.GlobalAccountID)
	if err != nil {
		return nil, errors.Wrapf(err, "while checking if instance is already created for global account %s", request.GlobalAccountID)
	}

	p.log.Infof("Found existing cls instance for global account %s", request.GlobalAccountID)

	if !exists {
		p.log.Infof("No cls instance found for global account %s", request.GlobalAccountID)
		return p.createNewInstance(smClient, request)
	}

	err = p.storage.Reference(instance.Version, instance.GlobalAccountID, request.SKRInstanceID)
	if err != nil {
		return nil, errors.Wrapf(err, "while deleting a cls instance for global account %s", request.GlobalAccountID)
	}

	p.log.Infof("Referencing the cls instance for global account %s by the skr %s", request.SKRInstanceID, request.GlobalAccountID)

	return &ProvisionResult{
		InstanceID:            instance.ID,
		ProvisioningTriggered: false,
		Region:                instance.Region,
	}, nil
}

func (p *provisioner) createNewInstance(smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance := internal.CLSInstance{
		ID:              uuid.New().String(),
		GlobalAccountID: request.GlobalAccountID,
		Region:          request.Region,
		CreatedAt:       time.Now(),
		SKRReferences:   []string{request.SKRInstanceID},
	}

	err := p.storage.InsertInstance(instance)
	if err != nil {
		return nil, errors.Wrapf(err, "while inserting a cls instance for global account %s", instance.GlobalAccountID)
	}

	p.log.Infof("Creating a new cls instance for global account %s", request.GlobalAccountID)

	err = p.creator.CreateInstance(smClient, servicemanager.InstanceKey{
		BrokerID:   request.BrokerID,
		ServiceID:  request.ServiceID,
		PlanID:     request.PlanID,
		InstanceID: instance.ID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "while deleting a cls instance for global account %s", request.GlobalAccountID)
	}

	return &ProvisionResult{
		InstanceID:            instance.ID,
		ProvisioningTriggered: true,
		Region:                request.Region,
	}, nil
}
