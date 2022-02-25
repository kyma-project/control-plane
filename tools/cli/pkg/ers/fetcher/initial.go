package fetcher

import (
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/ers/client"
)

type InitialFetcher struct {
	client    client.Client
	pageStart int
	pageSize  int
	pageLimit int
}

func NewInitialFetcher(client client.Client, pageStart, pageSize, pageLimit int) InstanceFetcher {
	return &InitialFetcher{client, pageStart, pageSize, pageLimit}
}

func (e InitialFetcher) GetAllInstances() ([]ers.Instance, error) {
	instances, err := e.client.GetPaged(e.pageStart, e.pageSize)
	output := make([]ers.Instance, 0)

	condition := func() bool {
		return err != nil || len(instances) > 0
	}

	page := e.pageStart
	for ok := condition(); ok; ok = condition() {
		page = page + 1

		if e.pageLimit != 0 && page > e.pageLimit {
			break
		}

		output = append(output, instances...)
		instances, err = e.client.GetPaged(page, e.pageSize)
	}

	return output, err
}

func (e InitialFetcher) GetInstanceById(id string) (*ers.Instance, error) {
	return e.client.GetOne(id)
}
