package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=DeprovisionerStorage --output=automock --outpkg=automock --case=underscore
type DeprovisionerStorage interface {
	RemoveInstance(globalAccountID string) error
	RemoveReference(globalAccountID, skrInstanceID string) (int, error)
}

type deprovisioner struct {
	storage DeprovisionerStorage
	log     logrus.FieldLogger
}

type DeprovisionRequest struct {
	GlobalAccountID string
	SKRInstanceID   string
	Instance        servicemanager.InstanceKey
}

func (d *deprovisioner) Deprovision(smClient servicemanager.Client, request *DeprovisionRequest) error {
	referencesLeft, err := d.storage.RemoveReference(request.GlobalAccountID, request.SKRInstanceID)
	if err != nil && !dberr.IsNotFound(err) {
		return errors.Wrapf(err, "while removing a reference to a cls instance for global account: %s and skr: %s", request.GlobalAccountID, request.SKRInstanceID)
	}

	if referencesLeft > 0 {
		return nil
	}

	if _, err = smClient.Deprovision(request.Instance, true); err != nil {
		return errors.Wrapf(err, "while deprovisioning a cls instance with ID %s for global account: %s", request.Instance.InstanceID, request.GlobalAccountID)
	}

	if err := d.storage.RemoveInstance(request.GlobalAccountID)

}
