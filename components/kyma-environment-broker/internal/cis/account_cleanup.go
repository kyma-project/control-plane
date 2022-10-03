package cis

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate mockery --name=CisClient --output=automock
type CisClient interface {
	FetchSubAccountsToDelete() ([]string, error)
}

//go:generate mockery --name=BrokerClient --output=automock
type BrokerClient interface {
	Deprovision(instance internal.Instance) (string, error)
}

type SubAccountCleanupService struct {
	client       CisClient
	brokerClient BrokerClient
	storage      storage.Instances
	log          logrus.FieldLogger
	chunksAmount int
}

func NewSubAccountCleanupService(client CisClient, brokerClient BrokerClient, storage storage.Instances, log logrus.FieldLogger) *SubAccountCleanupService {
	return &SubAccountCleanupService{
		client:       client,
		brokerClient: brokerClient,
		storage:      storage,
		log:          log,
		chunksAmount: 50,
	}
}

func (ac *SubAccountCleanupService) Run() error {
	subaccounts, err := ac.client.FetchSubAccountsToDelete()
	if err != nil {
		return errors.Wrap(err, "while fetching subaccounts by client")
	}

	subaccountsBatch := chunk(ac.chunksAmount, subaccounts)
	chunks := len(subaccountsBatch)
	errCh := make(chan error)
	done := make(chan struct{})
	var isDone bool

	for _, chunk := range subaccountsBatch {
		go ac.executeDeprovisioning(chunk, done, errCh)
	}

	for !isDone {
		select {
		case err := <-errCh:
			ac.log.Warnf("part of deprovisioning process failed with error: %s", err)
		case <-done:
			chunks--
			if chunks == 0 {
				isDone = true
			}
		}
	}

	ac.log.Info("SubAccount cleanup process finished")
	return nil
}

func (ac *SubAccountCleanupService) executeDeprovisioning(subaccounts []string, done chan<- struct{}, errCh chan<- error) {
	instances, err := ac.storage.FindAllInstancesForSubAccounts(subaccounts)
	if err != nil {
		errCh <- errors.Wrap(err, "while finding all instances by subaccounts")
		return
	}

	for _, instance := range instances {
		operation, err := ac.brokerClient.Deprovision(instance)
		if err != nil {
			errCh <- errors.Wrapf(err, "error occurred during deprovisioning instance with ID %s", instance.InstanceID)
			continue
		}
		ac.log.Infof("deprovisioning for instance %s (SubAccountID: %s) was triggered, operation: %s", instance.InstanceID, instance.SubAccountID, operation)
	}

	done <- struct{}{}
}

func chunk(amount int, data []string) [][]string {
	var divided [][]string

	for i := 0; i < len(data); i += amount {
		end := i + amount
		if end > len(data) {
			end = len(data)
		}
		divided = append(divided, data[i:end])
	}

	return divided
}
