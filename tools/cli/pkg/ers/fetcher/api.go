package fetcher

import "github.com/kyma-project/control-plane/tools/cli/pkg/ers"

type InstanceFetcher interface {
	GetAllInstances() ([]ers.Instance, error)
	GetInstanceById(id string) (*ers.Instance, error)
}
