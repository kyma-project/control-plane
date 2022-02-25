package client

import (
	"fmt"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/pkg/errors"
)

const environmentsPath = "%s/provisioning/v1/kyma/environments"
const brokersPath = "%s/provisioning/v1/brokers"
const pagedParams = "page=%d&size=%d"
const idParam = "id=%s"

type ersClient struct {
	url    string
	client *HTTPClient
}

func NewErsClient(url string, client *HTTPClient) Client {
	return &ersClient{
		url,
		client,
	}
}

func (c *ersClient) GetOne(instanceID string) (*ers.Instance, error) {
	instances, err := c.client.get(fmt.Sprintf(environmentsPath+"?"+idParam, c.url, instanceID))
	if err != nil {
		return nil, errors.Wrap(err, "Error while sending request")
	}

	if len(instances) != 1 {
		return nil, errors.New("Unexpectedly found multiple instances")
	}

	return &instances[0], nil
}

func (c *ersClient) GetPagedDefault() ([]ers.Instance, error) {
	return c.GetPaged(0, 1)
}

func (c *ersClient) GetPaged(pageNo, pageSize int) ([]ers.Instance, error) {
	return c.client.get(fmt.Sprintf(environmentsPath+"?"+pagedParams, c.url, pageNo, pageSize))
}

func (c *ersClient) Migrate(instanceID string) error {
	return c.client.put(fmt.Sprintf(environmentsPath+"/%s", c.url, instanceID))
}

func (c *ersClient) Switch(brokerID string) error {
	return c.client.put(fmt.Sprintf(brokersPath+"/%s", c.url, brokerID))
}
