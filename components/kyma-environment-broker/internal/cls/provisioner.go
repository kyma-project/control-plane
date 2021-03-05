package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=ProvisionerStorage --output=automock --outpkg=automock --case=underscore
type ProvisionerStorage interface {
	FindActiveByGlobalAccountID(globalAccountID string) (*internal.CLSInstance, bool, error)
	Insert(instance internal.CLSInstance) error
	Update(instance internal.CLSInstance) error
}

//go:generate mockery --name=InstanceCreator --output=automock --outpkg=automock --case=underscore
type InstanceCreator interface {
	CreateInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error
}

type provisioner struct {
	storage ProvisionerStorage
	creator InstanceCreator
}

func NewProvisioner(storage ProvisionerStorage, creator InstanceCreator) *provisioner {
	return &provisioner{
		storage: storage,
		creator: creator,
	}
}

type ProvisionRequest struct {
	GlobalAccountID string
	Region          string
	SKRInstanceID   string
	Instance        servicemanager.InstanceKey
}

type ProvisionResult struct {
	InstanceID string
	Region     string
}

func (p *provisioner) Provision(log logrus.FieldLogger, smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance, exists, err := p.storage.FindActiveByGlobalAccountID(request.GlobalAccountID)
	if err != nil {
		return nil, errors.Wrapf(err, "while checking if instance is already created for global account %s", request.GlobalAccountID)
	}

	if !exists {
		log.Infof("No cls instance found for global account %s", request.GlobalAccountID)
		return p.createNewInstance(log, smClient, request)
	}

	log.Infof("Found existing cls instance for global account %s", request.GlobalAccountID)

	instance.AddReference(request.SKRInstanceID)
	if err := p.storage.Update(*instance); err != nil {
		return nil, errors.Wrapf(err, "while updating a cls instance for global account %s", request.GlobalAccountID)
	}

	log.Infof("Referencing the cls instance for global account %s by the skr %s", request.GlobalAccountID, request.SKRInstanceID)

	return &ProvisionResult{
		InstanceID: instance.ID(),
		Region:     instance.Region(),
	}, nil
}

func (p *provisioner) createNewInstance(log logrus.FieldLogger, smClient servicemanager.Client, request *ProvisionRequest) (*ProvisionResult, error) {
	instance := internal.NewCLSInstance(request.GlobalAccountID, request.Region, internal.WithReferences(request.SKRInstanceID))

	err := p.storage.Insert(*instance)
	if err != nil {
		return nil, errors.Wrapf(err, "while inserting a cls instance for global account %s", instance.GlobalAccountID())
	}

	log.Infof("Creating a new cls instance for global account %s", request.GlobalAccountID)

	request.Instance.InstanceID = instance.ID()
	err = p.creator.CreateInstance(smClient, request.Instance)
	if err != nil {
		return nil, errors.Wrapf(err, "while creating a cls instance for global account %s", instance.GlobalAccountID())
	}

	return &ProvisionResult{
		InstanceID: instance.ID(),
		Region:     request.Region,
	}, nil
}
