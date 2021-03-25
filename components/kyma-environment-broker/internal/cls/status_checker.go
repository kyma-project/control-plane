package cls

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=StatusChecker --output=automock --outpkg=automock --case=underscore
type StatusChecker interface {
	CheckStatus(smClient servicemanager.Client, instanceKey servicemanager.InstanceKey) (ProvisionStatus, error)
}

type checker struct {
	storage       ProvisionerStorage
	statusChecker StatusChecker
}

func NewStatusChecker(storage ProvisionerStorage, statusChecker StatusChecker) *checker {
	return &checker{
		storage:       storage,
		statusChecker: statusChecker,
	}
}

func (p *checker) CheckProvisionStatus(smClient servicemanager.Client, instanceKey servicemanager.InstanceKey, log logrus.FieldLogger) (ProvisionStatus, error) {
	res, err := p.statusChecker.CheckStatus(smClient, instanceKey)
	if err != nil {
		switch res {
		case Failed:
			log.Infof("Deleting the CLS instance from DB: %v", instanceKey.InstanceID)
			err = p.storage.Delete(instanceKey.InstanceID)
			if err != nil {
				log.Warnf("Unable to delete CLS Instance from DB: %v", instanceKey.InstanceID)
				return Retry, err
			}
			return Failed, err
		case Retry:
			return Retry, err
		}
	}
	return res, err
}
