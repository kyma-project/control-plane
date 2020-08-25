package e2e

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pkg/errors"
)

type FakeBrokerClient struct {
	storage storage.Instances
}

func NewFakeBrokerClient(storage storage.Instances) *FakeBrokerClient {
	return &FakeBrokerClient{storage: storage}
}

func (fbc *FakeBrokerClient) Deprovision(instance internal.Instance) (string, error) {
	err := fbc.storage.Delete(instance.InstanceID)
	if err != nil {
		return "", errors.Wrapf(err, "fake broker client cannot remove instance with ID: %s", instance.InstanceID)
	}

	return "061ced07-6225-43ea-b574-03ca78d5b1bc", nil
}
