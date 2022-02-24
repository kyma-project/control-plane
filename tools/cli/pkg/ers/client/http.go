package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
)

type HttpClient struct {
	logger logger.Logger
	client *http.Client
}

func NewHttpClient(logger logger.Logger, client *http.Client) *HttpClient {
	return &HttpClient{
		logger,
		client,
	}
}

func (c *HttpClient) put(url string) error {
	c.do(nil, func() (resp *http.Response, err error) {
		req, err := http.NewRequest("PUT", url, nil)
		if err != nil {
			return nil, err
		}
		return c.client.Do(req)
	})

	return nil
}

func (c *HttpClient) get(url string) ([]ers.Instance, error) {
	kymaEnv := make([]ers.Instance, 0)

	err := c.do(&kymaEnv, func() (resp *http.Response, err error) {
		return c.client.Get(url)
	})

	return kymaEnv, err
}

func (c *HttpClient) do(v interface{}, request func() (resp *http.Response, err error)) error {
	resp, err := request()

	c.logger.Debugf("Sending request to %s", resp.Request.URL)

	if err != nil {
		return errors.Wrap(err, "Error while sending request")
	}

	c.logger.Debugf("Response status: %s", resp.Status)

	defer func() {
		resp.Body.Close()
	}()

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
