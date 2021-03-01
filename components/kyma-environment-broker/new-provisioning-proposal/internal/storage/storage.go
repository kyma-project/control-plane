package storage

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/new-provisioning-proposal/internal/storage/driver/memory"

type BrokerStorage interface {
	Operations() Operations
	Provisioning() Provisioning
}

type storage struct {
	operation Operations
}

func (s storage) Operations() Operations {
	return s.operation
}

func (s storage) Provisioning() Provisioning {
	return s.operation
}

func NewMemoryStorage() BrokerStorage {
	return storage{
		operation: memory.NewOperation(),
	}
}
