package cls

import (
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ProvisionStatus string

const (
	InProgress ProvisionStatus = "in progress"
	Succeeded  ProvisionStatus = "succeeded"
	Retry      ProvisionStatus = "retry"
	Failed     ProvisionStatus = "failed"
)

type checker struct {
	storage ProvisionerStorage
}

func NewStatusChecker(storage ProvisionerStorage) *checker {
	return &checker{
		storage: storage,
	}
}

func (c *checker) CheckProvisionStatus(smClient servicemanager.Client, instanceKey servicemanager.InstanceKey, log logrus.FieldLogger) (ProvisionStatus, error) {
	res, err := c.checkStatus(smClient, instanceKey)
	if err != nil {
		if res == Failed {
			err = c.storage.Delete(instanceKey.InstanceID)
			if err != nil {
				return Retry, errors.Wrapf(err, "while deleting CLS instance %s", instanceKey.InstanceID)
			}
			return Failed, err
		}

		if res == Retry {
			return Retry, err
		}
	}

	return res, nil
}

func (c *checker) checkStatus(smClient servicemanager.Client, instanceKey servicemanager.InstanceKey) (ProvisionStatus, error) {
	resp, err := smClient.LastInstanceOperation(instanceKey, "")
	if err != nil {
		if kebError.IsTemporaryError(err) {
			return Retry, err
		}
		return Failed, err
	}

	switch resp.State {
	case servicemanager.InProgress:
		return InProgress, nil
	case servicemanager.Failed:
		return Failed, err
	}
	return Succeeded, nil
}
