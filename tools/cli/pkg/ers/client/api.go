package client

import (
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
)

type Client interface {
	GetOne(id string) (*ers.Instance, error)
	GetPaged(pageStart, pageSize int) ([]ers.Instance, error)

	Migrate(instanceID string) error
	Switch(brokerID string) error
	// Handle status codes and how we should react to them - 4xx, 5xx errors

	Close()
}
