package mothership

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

//go:generate mockgen -destination=automock/client.go -package=automock . HttpClient

var (
	ErrMothershipResponse = errors.New("reconciler error response")
)

type Client interface {
	List(ctx context.Context, filters map[string]string) ([]Reconciliation, error)
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type client struct {
	HttpClient
	URLProvider
}

func (c *client) List(ctx context.Context, filters map[string]string) ([]Reconciliation, error) {

	listEndpoint := c.Provide(EndpointReconcile, filters)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listEndpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if isErrResponse(resp.StatusCode) {
		err := responseErr(resp)
		return nil, err
	}

	var result []Reconciliation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.WithStack(ErrMothershipResponse)
	}

	return result, err
}

func isErrResponse(statusCode int) bool {
	return statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices
}

func responseErr(resp *http.Response) error {
	var msg string
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		msg = "unknown error"
	}
	return errors.Wrapf(ErrMothershipResponse, "%s %d", msg, resp.StatusCode)
}

func NewClient(mothershipURL string) (Client, error) {

	u, err := url.Parse(mothershipURL)
	if err != nil {
		return nil, err
	}

	return &client{
		HttpClient: http.DefaultClient,
		URLProvider: urlProvider{
			mothershipURL: *u,
		},
	}, err
}
