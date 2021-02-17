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
	MarkAsBeingRemoved(version int, globalAccountID string) error
	RemoveInstance(version int, globalAccountID string) error
}

//go:generate mockery --name=InstanceRemover --output=automock --outpkg=automock --case=underscore
type InstanceRemover interface {
	RemoveInstance(smClient servicemanager.Client, instance servicemanager.InstanceKey) error
}

type Deprovisioner struct {
	config  *Config
	storage DeprovisionerStorage
	remover InstanceRemover
	log     logrus.FieldLogger
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
	for _, ref := range instance.SKRReferences {
		if ref == request.SKRInstanceID {
			isReferenced = true
		}
	}
	if !isReferenced {
		d.log.Warnf("Provided CLS instance for global account %s is not referenced by the SKR %s", request.GlobalAccountID, request.SKRInstanceID)
		return nil
	}

	if len(instance.SKRReferences) > 1 {
		if err := d.storage.Unreference(instance.Version, request.GlobalAccountID, request.SKRInstanceID); err != nil {
			return errors.Wrapf(err, "while unreferencing a cls instance for global account: %s", request.GlobalAccountID)
		}

		return nil
	}

	if err := d.storage.MarkAsBeingRemoved(instance.Version, request.GlobalAccountID); err != nil {
		return errors.Wrapf(err, "while marking a cls instance as being removed for global account: %s", request.GlobalAccountID)
	}

	if err := d.remover.RemoveInstance(smClient, request.Instance); err != nil {
		return errors.Wrapf(err, "while deleting a cls instance for global account: %s", request.GlobalAccountID)
	}

	return nil
}
