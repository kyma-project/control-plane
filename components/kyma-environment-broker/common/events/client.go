package events

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type EventLevel string

const (
	InfoEventLevel  EventLevel = "info"
	ErrorEventLevel EventLevel = "error"
)

type EventDTO struct {
	ID          string
	Level       EventLevel
	InstanceID  *string
	OperationID *string
	Message     string
	CreatedAt   time.Time
}

type EventFilter struct {
	InstanceIDs  []string
	OperationIDs []string
}

// Client is the interface to interact with the KEB /events API as an HTTP client using OIDC ID token in JWT format.
type Client interface {
	ListEvents(instanceIDs []string) ([]EventDTO, error)
}

type client struct {
	url        string
	httpClient *http.Client
}

// NewClient constructs and returns new Client for KEB /events API
// It takes the following arguments:
//   - url        : base url of all KEB APIs, e.g. https://kyma-env-broker.kyma.local
//   - httpClient : underlying HTTP client used for API call to KEB
func NewClient(url string, httpClient *http.Client) Client {
	return &client{
		url:        url,
		httpClient: httpClient,
	}
}

// ListEvents
func (c *client) ListEvents(instanceIDs []string) ([]EventDTO, error) {
	var events []EventDTO
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/events", c.url), nil)
	if err != nil {
		return events, fmt.Errorf("while creating request: %v", err)
	}
	q := req.URL.Query()
	q.Add("instance_ids", strings.Join(instanceIDs, ","))
	req.URL.RawQuery = q.Encode()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return events, fmt.Errorf("while calling %s: %v", req.URL.String(), err)
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
		return events, fmt.Errorf("calling %s returned %d (%s) status", req.URL.String(), resp.StatusCode, resp.Status)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&events)
	if err != nil {
		return events, fmt.Errorf("while decoding response body: %v", err)
	}
	return events, nil
}

func drainResponseBody(body io.Reader) error {
	if body == nil {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, 4096))
	return err
}
