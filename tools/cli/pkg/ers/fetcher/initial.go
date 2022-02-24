package fetcher

import (
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
)

type InitialFetcher struct {
	client client.Client
}

func NewInitialFetcher(client client.Client) InstanceFetcher {
	return &InitialFetcher{client}
}

func (e InitialFetcher) GetAllInstances() ([]ers.Instance, error) {
	page := 0
	pageSize := 5

	instances, err := e.client.GetPaged(page, pageSize)
	output := make([]ers.Instance, 0)

	condition := func() bool {
		return err != nil || len(instances) > 0
	}

	for ok := condition(); ok; ok = condition() {
		page = page + 1
		output = append(output, instances...)
		instances, err = e.client.GetPaged(page, pageSize)
	}

	return output, err
}

func (e InitialFetcher) GetInstanceById(id string) (*ers.Instance, error) {
	return e.client.GetOne(id)
}
