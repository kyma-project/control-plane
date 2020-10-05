package client

import (
	"context"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// Client is the interface to interact with the kubeconfig-service as an HTTP client using OIDC ID token in JWT format.
type Client interface {
	GetKubeConfig(tenantID, runtimeID string) (string, error)
}

type client struct {
	url        string
	httpClient *http.Client
}

// NewClient constructs and returns new Client for kubeconfig-service
// It takes the following arguments:
//   - ctx  : context in which the http request will be executed
//   - url  : base url of the kubeconfig-service API
//   - auth : TokenSource object which provides the ID token for the HTTP request
func NewClient(ctx context.Context, url string, auth oauth2.TokenSource) Client {
	c := &client{
		url:        url,
		httpClient: oauth2.NewClient(ctx, auth),
	}
	return c
}

// GetKubeConfig
func (c *client) GetKubeConfig(tenantID, runtimeID string) (string, error) {
	url := fmt.Sprintf("%s/kubeconfig/%s/%s", c.url, tenantID, runtimeID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", errors.Wrapf(err, "while calling %s", url)
	}

	// Drain response body and close, return error to context if there isn't any.
	defer func() {
		derr := drainResponseBody(resp.Body)
		if err == nil {
			err = derr
		}
		cerr := resp.Body.Close()
		if err == nil {
			err = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("calling %s returned %s status", url, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "while reading response body")
	}
	return html.UnescapeString(string(body)), nil
}

func drainResponseBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}
