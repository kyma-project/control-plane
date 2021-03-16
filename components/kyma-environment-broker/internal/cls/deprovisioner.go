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
	Update(instance internal.CLSInstance) error
	Delete(clsInstanceID string) error
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
	instance, exists, err := d.storage.FindByID(request.Instance.InstanceID)
	if err != nil {
		return errors.Wrapf(err, "while finding CLS instance %s", request.Instance.InstanceID)
	}

	if !exists {
		return nil
	}

	if !instance.IsReferencedBy(request.SKRInstanceID) {
		d.log.Warnf("Provided CLS instance %s is not referenced by the SKR %s", instance.ID, request.SKRInstanceID)
		return nil
	}

	if err := instance.RemoveReference(request.SKRInstanceID); err != nil {
		return errors.Wrapf(err, "while unreferencing CLS instance %s", instance.ID())
	}

	if err := d.storage.Update(*instance); err != nil {
		return errors.Wrapf(err, "while updating CLS instance %s", instance.ID())
	}

	if instance.IsBeingRemoved() {
		d.log.Infof("Removing CLS instance %s", instance.ID)

		if err := d.remover.RemoveInstance(smClient, request.Instance); err != nil {
			return errors.Wrapf(err, "while removing CLS instance %s", instance.ID())
		}

		if err := d.storage.Delete(instance.ID()); err != nil {
			return errors.Wrapf(err, "while deleting CLS instance %s", instance.ID())
		}
	}

	return nil
}
