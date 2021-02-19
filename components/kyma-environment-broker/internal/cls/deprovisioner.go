package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=DeprovisionerStorage --output=automock --outpkg=automock --case=underscore
type DeprovisionerStorage interface {
	FindInstance(globalAccountID string) (*internal.CLSInstance, bool, error)
	Unreference(version int, globalAccountID, skrInstanceID string) error
	MarkAsBeingRemoved(version int, globalAccountID, skrInstanceID string) error
	RemoveInstance(version int, globalAccountID string) error
}

//go:generate mockery --name=InstanceRemover --output=automock --outpkg=automock --case=underscore
type InstanceRemover interface {
	RemoveInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error
}

type Deprovisioner struct {
	storage DeprovisionerStorage
	remover InstanceRemover
	log     logrus.FieldLogger
}

func NewDeprovisioner(storage DeprovisionerStorage, remover InstanceRemover, log logrus.FieldLogger) *Deprovisioner {
	return &Deprovisioner{
		storage: storage,
		remover: remover,
		log:     log,
	}
}

type DeprovisionRequest struct {
	GlobalAccountID string
	SKRInstanceID   string
	Instance        servicemanager.InstanceKey
}

func (d *Deprovisioner) Deprovision(smClient servicemanager.Client, request *DeprovisionRequest) error {
	instance, _, err := d.storage.FindInstance(request.GlobalAccountID)
	if err != nil {
		return errors.Wrapf(err, "while trying to lookup an instance for global account: %s", request.GlobalAccountID)
	}

	isReferenced := false
	for _, ref := range instance.ReferencedSKRInstanceIDs {
		if ref == request.SKRInstanceID {
			isReferenced = true
		}
	}
	if !isReferenced {
		d.log.Warnf("Provided cls instance for global account %s is not referenced by the skr %s", request.GlobalAccountID, request.SKRInstanceID)
		return nil
	}

	if len(instance.ReferencedSKRInstanceIDs) > 1 {
		if err := d.storage.Unreference(instance.Version, request.GlobalAccountID, request.SKRInstanceID); err != nil {
			return errors.Wrapf(err, "while unreferencing a cls instance for global account %s", request.GlobalAccountID)
		}

		d.log.Infof("Unreferenced the skr %s from the cls instance for global account %s", request.SKRInstanceID, request.GlobalAccountID)
		return nil
	}

	d.log.Infof("Marking the cls instance for global account %s as being removed by skr %s", request.GlobalAccountID, request.SKRInstanceID)

	if err := d.storage.MarkAsBeingRemoved(instance.Version, request.GlobalAccountID, request.SKRInstanceID); err != nil {
		return errors.Wrapf(err, "while marking a cls instance as being removed for global account %s", request.GlobalAccountID)
	}

	d.log.Infof("Removing the cls instance for global account %s", request.GlobalAccountID)

	if err := d.remover.RemoveInstance(smClient, request.Instance); err != nil {
		return errors.Wrapf(err, "while removing a cls instance for global account %s", request.GlobalAccountID)
	}

	if err := d.storage.RemoveInstance(instance.Version, request.GlobalAccountID); err != nil {
		return errors.Wrapf(err, "while removing a cls instance record for global account %s", request.GlobalAccountID)
	}

	return nil
}
