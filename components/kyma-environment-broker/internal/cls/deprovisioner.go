package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=DeprovisionerStorage --output=automock --outpkg=automock --case=underscore
type DeprovisionerStorage interface {
	FindByID(clsInstanceID string) (*internal.CLSInstance, bool, error)
	Unreference(version int, clsInstanceID, skrInstanceID string) error
	MarkAsBeingRemoved(version int, clsInstanceID, skrInstanceID string) error
	Remove(clsInstanceID string) error
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
	SKRInstanceID string
	Instance      servicemanager.InstanceKey
}

func (d *Deprovisioner) Deprovision(smClient servicemanager.Client, request *DeprovisionRequest) error {
	instance, _, err := d.storage.FindByID(request.Instance.InstanceID)
	if err != nil {
		return errors.Wrapf(err, "while trying to find the cls instance %s", request.Instance.InstanceID)
	}

	isReferenced := false
	for _, ref := range instance.ReferencedSKRInstanceIDs {
		if ref == request.SKRInstanceID {
			isReferenced = true
		}
	}
	if !isReferenced {
		d.log.Warnf("Provided cls instance %s is not referenced by the skr %s", instance.ID, request.SKRInstanceID)
		return nil
	}

	if len(instance.ReferencedSKRInstanceIDs) > 1 {
		if err := d.storage.Unreference(instance.Version, instance.ID, request.SKRInstanceID); err != nil {
			return errors.Wrapf(err, "while unreferencing the cls instance %s", instance.ID)
		}

		d.log.Infof("Unreferenced the skr %s from the cls instance %s", request.SKRInstanceID, instance.ID)
		return nil
	}

	d.log.Infof("Marking the cls instance %s as being removed by the skr %s", instance.ID, request.SKRInstanceID)

	if err := d.storage.MarkAsBeingRemoved(instance.Version, instance.ID, request.SKRInstanceID); err != nil {
		return errors.Wrapf(err, "while marking a cls instance %s as being removed", instance.ID)
	}

	d.log.Infof("Removing the cls instance %s", instance.ID)

	if err := d.remover.RemoveInstance(smClient, request.Instance); err != nil {
		return errors.Wrapf(err, "while removing the cls instance %s", instance.ID)
	}

	if err := d.storage.Remove(instance.ID); err != nil {
		return errors.Wrapf(err, "while removing the cls instance record %s", instance.ID)
	}

	return nil
}
