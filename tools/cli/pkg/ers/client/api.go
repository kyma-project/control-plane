package client

import "github.com/kyma-project/control-plane/tools/cli/pkg/ers"

type Client interface {
	GetOne(id string) (*ers.Instance, error)
	GetPagedDefault() ([]ers.Instance, error)
	GetPaged(pageNo, pageSize int) ([]ers.Instance, error)

	Migrate(instanceId string) error
	Switch(brokerId string) error
	// Handle status codes and how we should react to them - 4xx, 5xx errors
}
