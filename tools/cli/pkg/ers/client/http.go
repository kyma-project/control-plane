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
	"golang.org/x/oauth2/clientcredentials"
)

const timeoutInMilli = 3000

type HTTPClient struct {
	logger logger.Logger
	Client *http.Client
}

func NewHTTPClient(logger logger.Logger) (*HTTPClient, error) {

	// create a shared ERS HTTP client which does the oauth flow
	client, err := createConfigClient()
	if err != nil {
		return nil, errors.Wrap(err, "while create http client")
	}

	return &HTTPClient{
		logger,
		client,
	}, nil
}

func (c *HTTPClient) put(url string) error {
	return c.do(nil, func(ctx context.Context) (resp *http.Response, err error) {
		c.logger.Debugf("Sending request to %s", url)
		req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
		if err != nil {
			return nil, errors.Wrap(err, "Error while sending a PUT request")
		}
		return c.Client.Do(req)
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
		return c.Client.Do(req)
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

func (c *HTTPClient) Close() {
	c.Client.CloseIdleConnections()
}

func createConfigClient() (*http.Client, error) {
	if ers.GlobalOpts.ClientID() == "" ||
		ers.GlobalOpts.ClientSecret() == "" ||
		ers.GlobalOpts.OauthUrl() == "" {
		return nil, errors.New("no auth data provided")
	}

	config := clientcredentials.Config{
		ClientID:     ers.GlobalOpts.ClientID(),
		ClientSecret: ers.GlobalOpts.ClientSecret(),
		TokenURL:     ers.GlobalOpts.OauthUrl(),
	}
	return config.Client(context.Background()), nil
}
