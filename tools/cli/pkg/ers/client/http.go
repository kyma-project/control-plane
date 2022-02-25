package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
)

type HTTPClient struct {
	logger logger.Logger
	client *http.Client
}

func NewHTTPClient(logger logger.Logger, client *http.Client) *HTTPClient {
	return &HTTPClient{
		logger,
		client,
	}
}

func (c *HTTPClient) put(url string) error {
	return c.do(nil, func() (resp *http.Response, err error) {
		req, err := http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Error while sending a PUT request")
		}
		return c.client.Do(req)
	})
}

func (c *HTTPClient) get(url string) ([]ers.Instance, error) {
	kymaEnv := make([]ers.Instance, 0)

	err := c.do(&kymaEnv, func() (resp *http.Response, err error) {
		return c.client.Get(url)
	})

	return kymaEnv, err
}

func (c *HTTPClient) do(v interface{}, request func() (resp *http.Response, err error)) error {
	resp, err := request()

	c.logger.Debugf("Sending request to %s", resp.Request.URL)

	if err != nil {
		return errors.Wrap(err, "Error while sending request")
	}

	c.logger.Debugf("Response status: %s", resp.Status)

	defer resp.Body.Close()

	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Error while reading from response")
	}
	c.logger.Debug("Received raw response: %s", string(d))

	if v == nil {
		return nil
	}

	err = json.Unmarshal(d, v)
	if err != nil {
		return errors.Wrap(err, "Error while unmarshaling")
	}

	return nil
}
