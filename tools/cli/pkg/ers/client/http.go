package client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/tools/cli/pkg/ers"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
	"github.com/pkg/errors"
)

const timeoutInMilli = 400

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
	return c.do(nil, func(ctx context.Context) (resp *http.Response, err error) {
		c.logger.Debugf("Sending request to %s", url)
		req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Error while sending a PUT request")
		}
		return c.client.Do(req)
	})
}

func (c *HTTPClient) get(url string) ([]ers.Instance, error) {
	kymaEnv := make([]ers.Instance, 0)

	err := c.do(&kymaEnv, func(ctx context.Context) (resp *http.Response, err error) {
		c.logger.Debugf("Sending request to %s", url)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		return c.client.Do(req)
	})

	return kymaEnv, err
}

func (c *HTTPClient) do(v interface{}, request func(ctx context.Context) (resp *http.Response, err error)) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutInMilli)*time.Millisecond)
	defer cancel()

	resp, err := request(ctx)

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
