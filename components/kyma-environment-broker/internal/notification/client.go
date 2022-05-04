package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/pkg/errors"
)

const (
	PathCreateEvent             string = "/createMaintenanceEvent"
	PathUpdateEvent             string = "/updateMaintenanceEvent"
	PathCancelEvent             string = "/cancelMaintenanceEvent"
	KubernetesMaintenanceNumber string = "0"
	KymaMaintenanceNumber       string = "1"
	UnderMaintenanceEventState  string = "1"
	FinishedMaintenanceState    string = "2"
	CancelledMaintenanceState   string = "3"
)

type (
	ClientConfig struct {
		URL string
	}

	Client struct {
		config     ClientConfig
		httpClient *http.Client
	}

	Request struct {
		Method  string
		Path    string
		Body    io.Reader
		Headers map[string]string
		Delete  bool
	}

	NotificationTenant struct {
		InstanceID string `json:"instanceId"`
		StartDate  string `json:"startDateTime,omitempty"`
		EndDate    string `json:"endDateTime,omitempty"`
		State      string `json:"eventState,omitempty"`
	}

	CreateEventRequest struct {
		OrchestrationID string               `json:"orchestrationId"`
		EventType       string               `json:"eventType"`
		Tenants         []NotificationTenant `json:"tenants"`
	}

	UpdateEventRequest struct {
		OrchestrationID string               `json:"orchestrationId"`
		Tenants         []NotificationTenant `json:"tenants"`
	}

	CancelEventRequest struct {
		OrchestrationID string `json:"orchestrationId"`
	}
)

func NewClient(cli *http.Client, cfg ClientConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: cli,
	}
}

func (c *Client) CreateEvent(payload CreateEventRequest) error {
	return c.callPost(PathCreateEvent, payload)
}

func (c *Client) UpdateEvent(payload UpdateEventRequest) error {
	return c.callPatch(PathUpdateEvent, payload)
}

func (c *Client) CancelEvent(payload CancelEventRequest) error {
	return c.callPatch(PathCancelEvent, payload)
}

func (c *Client) callPatch(path string, payload interface{}) (err error) {
	request, err := c.jsonRequest(path, http.MethodPatch, payload)
	if err != nil {
		return errors.Wrapf(err, "while creating json request for path %s", path)
	}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for call method")
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while making request for path %s", path)
	}

	return nil
}

func (c *Client) callPost(path string, payload interface{}) (err error) {
	request, err := c.jsonRequest(path, http.MethodPost, payload)
	if err != nil {
		return errors.Wrapf(err, "while creating json request for path %s", path)
	}

	response, err := c.do(request)
	defer func() {
		if closeErr := c.closeResponseBody(response); closeErr != nil {
			err = kebError.AsTemporaryError(closeErr, "while closing response body for call method")
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while making request for path %s", path)
	}

	return nil
}

func (c *Client) jsonRequest(path string, method string, payload interface{}) (*Request, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(payload)
	if err != nil {
		return &Request{}, err
	}

	return &Request{
		Method:  method,
		Path:    path,
		Body:    buffer,
		Headers: map[string]string{"content-type": "application/json"},
	}, nil
}

func (c *Client) do(sciReq *Request) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.config.URL, sciReq.Path)
	req, err := http.NewRequest(sciReq.Method, url, sciReq.Body)
	if err != nil {
		return nil, err
	}

	req.Close = true
	for h, v := range sciReq.Headers {
		req.Header.Set(h, v)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return &http.Response{}, kebError.AsTemporaryError(err, "while making request")
	}

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return response, nil
	case http.StatusNotFound:
		if sciReq.Delete {
			return response, nil
		}
	case http.StatusRequestTimeout:
		return response, kebError.NewTemporaryError(c.responseErrorMessage(response))
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return response, kebError.NewTemporaryError(c.responseErrorMessage(response))
	}
	return response, errors.Errorf("while sending request to IAS: %s", c.responseErrorMessage(response))
}

func (c *Client) closeResponseBody(response *http.Response) error {
	if response == nil {
		return nil
	}
	if response.Body == nil {
		return nil
	}
	return response.Body.Close()
}

func (c *Client) responseErrorMessage(response *http.Response) string {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Sprintf("unexpected status code %d cannot read body response: %s", response.StatusCode, err)
	}
	return fmt.Sprintf("unexpected status code %d with body: %s", response.StatusCode, string(body))
}
