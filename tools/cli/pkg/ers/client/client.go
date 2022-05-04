package client

import (
	"fmt"

	"errors"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
)

const environmentsPath = "%s/provisioning/v1/kyma/environments"
const brokersPath = "%s/provisioning/v1/kyma/brokers"
const pagedParams = "page=%d&size=%d"
const idParam = "id=%s"

type ersClient struct {
	url    string
	Client *HTTPClient
}

func NewErsClient() (Client, error) {
	url := ers.GlobalOpts.ErsUrl()
	logger := logger.New()
	client, err := NewHTTPClient(logger)
	if err != nil {
		return nil, fmt.Errorf("while ers client creation: %w", err)
	}

	return &ersClient{
		url,
		client,
	}, nil
}

func (c *ersClient) GetOne(instanceID string) (*ers.Instance, error) {
	instances, err := c.Client.get(fmt.Sprintf(environmentsPath+"?"+idParam, c.url, instanceID))
	if err != nil {
		return nil, fmt.Errorf("while sending request: %w", err)
	}

	if len(instances) != 1 {
		return nil, errors.New("Unexpectedly found multiple instances")
	}

	return &instances[0], nil
}

func (c *ersClient) GetPaged(pageNo, pageSize int) ([]ers.Instance, error) {
	return c.Client.get(fmt.Sprintf(environmentsPath+"?"+pagedParams, c.url, pageNo, pageSize))
}

func (c *ersClient) Migrate(instanceID string) error {
	return c.Client.put(fmt.Sprintf(environmentsPath+"/%s", c.url, instanceID))
}

func (c *ersClient) Switch(brokerID string) error {
	return c.Client.put(fmt.Sprintf(brokersPath+"/%s", c.url, brokerID))
}

func (c *ersClient) Close() {
	c.Client.Close()
}
