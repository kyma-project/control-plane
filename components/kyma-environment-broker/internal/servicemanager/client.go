package servicemanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Peripli/service-manager-cli/pkg/query"
	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/iosafety"
	"github.com/pkg/errors"
)

type (
	client struct {
		creds      Credentials
		httpClient *http.Client
	}
)

func New(credentials Credentials) Client {
	return &client{
		creds:      credentials.WithNormalizedURL(),
		httpClient: http.DefaultClient,
	}
}

func NewWithHttpClient(credentials Credentials, httpClient *http.Client) Client {
	return &client{
		creds:      credentials.WithNormalizedURL(),
		httpClient: httpClient,
	}
}

func (c *client) ListOfferingsByName(name string) (*types.ServiceOfferings, error) {
	offerings := &types.ServiceOfferings{}
	err := c.get(web.ServiceOfferingsURL, offerings, &query.Parameters{
		FieldQuery: []string{fmt.Sprintf("name eq '%s'", name)},
	})
	if err != nil {
		return nil, err
	}
	return offerings, nil
}

func (c *client) ListOfferings() (*types.ServiceOfferings, error) {
	offerings := &types.ServiceOfferings{}
	err := c.get(web.ServiceOfferingsURL, offerings, nil)
	if err != nil {
		return nil, err
	}
	return offerings, nil
}

func (c *client) ListPlansByName(planName, offeringID string) (*types.ServicePlans, error) {
	result := &types.ServicePlans{}
	err := c.get(web.ServicePlansURL, result,
		&query.Parameters{
			FieldQuery: []string{
				fmt.Sprintf("name eq '%s'", planName),
				fmt.Sprintf("service_offering_id eq '%s'", offeringID),
			},
		})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) Provision(brokerID string, input ProvisioningInput, acceptsIncomplete bool) (*ProvisionResponse, error) {
	url := fmt.Sprintf("%s/%s/v2/service_instances/%s", web.OSBURL, brokerID, input.ID)

	body, err := json.Marshal(input)
	if err != nil {
		return nil, errors.Wrapf(err, "while encoding input body")
	}
	bufferedBody := bytes.NewBuffer(body)

	q := &query.Parameters{}
	c.configureAcceptsIncompleteParam(q, acceptsIncomplete)

	response := &ProvisionResponseBody{}
	statusCode, err := c.call(http.MethodPut, url, response, q, bufferedBody)
	if err != nil {
		return nil, errors.Wrapf(err, "while calling provision endpoint")
	}
	return &ProvisionResponse{
		ProvisionResponseBody: *response,
		HTTPResponse:          HTTPResponse{StatusCode: statusCode},
	}, nil
}

func (c *client) configureAcceptsIncompleteParam(q *query.Parameters, acceptsIncomplete bool) {
	if acceptsIncomplete {
		q.GeneralParams = append(q.GeneralParams, "accepts_incomplete=true")
	}
}

func (c *client) Deprovision(key InstanceKey, acceptsIncomplete bool) (*DeprovisionResponse, error) {
	url := fmt.Sprintf("%s/%s/v2/service_instances/%s", web.OSBURL, key.BrokerID, key.InstanceID)
	q := &query.Parameters{
		GeneralParams: []string{
			fmt.Sprintf("service_id=%s", key.ServiceID),
			fmt.Sprintf("plan_id=%s", key.PlanID),
		},
	}
	c.configureAcceptsIncompleteParam(q, acceptsIncomplete)

	response := OperationResponse{}
	statusCode, err := c.call(http.MethodDelete, url, &response, q, nil)
	if statusCode == http.StatusGone {
		err = nil
	}

	return &DeprovisionResponse{
		OperationResponse: response,
		HTTPResponse:      HTTPResponse{StatusCode: statusCode},
	}, err
}

func (c *client) Unbind(instanceKey InstanceKey, bindingID string, acceptsIncomplete bool) (*DeprovisionResponse, error) {
	url := fmt.Sprintf("%s/%s/v2/service_instances/%s/service_bindings/%s", web.OSBURL, instanceKey.BrokerID, instanceKey.InstanceID, bindingID)
	q := &query.Parameters{
		GeneralParams: []string{
			fmt.Sprintf("service_id=%s", instanceKey.ServiceID),
			fmt.Sprintf("plan_id=%s", instanceKey.PlanID),
		}}
	c.configureAcceptsIncompleteParam(q, acceptsIncomplete)

	response := OperationResponse{}
	statusCode, err := c.call(http.MethodDelete, url, &response, q, nil)
	if statusCode == http.StatusGone {
		// the HTTP 410 GONE does not mean it was an error, the spec says:
		// "MUST be returned if the Service Binding does not exist."
		err = nil
	}

	return &DeprovisionResponse{
		OperationResponse: response,
		HTTPResponse:      HTTPResponse{StatusCode: statusCode},
	}, err
}

func (c *client) Bind(instanceKey InstanceKey, bindingID string, parameters interface{}, acceptsIncomplete bool) (*BindingResponse, error) {
	req := &struct {
		ServiceID  string                 `json:"service_id"`
		PlanID     string                 `json:"plan_id"`
		Parameters interface{}            `json:"parameters"`
		Context    map[string]interface{} `json:"context"`
	}{
		ServiceID:  instanceKey.ServiceID,
		PlanID:     instanceKey.PlanID,
		Parameters: parameters,
		Context: map[string]interface{}{
			"platform": "kubernetes",
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "while marshaling binding body")
	}
	bufferedBody := bytes.NewBuffer(body)

	path := fmt.Sprintf("%s/%s/v2/service_instances/%s/service_bindings/%s", web.OSBURL, instanceKey.BrokerID, instanceKey.InstanceID, bindingID)

	q := &query.Parameters{}
	c.configureAcceptsIncompleteParam(q, acceptsIncomplete)

	resp := Binding{}
	statusCode, err := c.call(http.MethodPut, path, &resp, nil, bufferedBody)

	return &BindingResponse{
		Binding: resp,
		HTTPResponse: HTTPResponse{
			StatusCode: statusCode,
		},
	}, err
}

func (c *client) GetBinding(instanceKey InstanceKey, bindingID string) (*types.ServiceBinding, error) {
	path := fmt.Sprintf("%s/%s/v2/service_bindings/%s", web.OSBURL, instanceKey.BrokerID, bindingID)
	resp := types.ServiceBinding{}
	//TODO: check status code too?
	_, err := c.call(http.MethodGet, path, &resp, nil, bytes.NewBuffer([]byte{}))
	return &resp, err
}

func (c *client) LastInstanceOperation(key InstanceKey, operationID string) (LastOperationResponse, error) {
	resp := LastOperationResponse{}
	path := fmt.Sprintf("%s/%s/v2/service_instances/%s/last_operation", web.OSBURL, key.BrokerID, key.InstanceID)
	q := &query.Parameters{
		GeneralParams: []string{"accepts_incomplete=true",
			fmt.Sprintf("service_id=%s", key.ServiceID),
			fmt.Sprintf("plan_id=%s", key.PlanID),
		},
	}
	if operationID != "" {
		q.GeneralParams = append(q.GeneralParams, fmt.Sprintf("operation=%s", operationID))
	}
	err := c.get(path, &resp, q)
	return resp, err
}

func (c *client) createHTTPRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.creds.Username, c.creds.Password)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Broker-API-Version", "2.16")
	return req, err
}

func (c *client) call(method, path string, response interface{}, q *query.Parameters, body io.Reader) (int, error) {
	params := q.Encode()
	url := c.creds.URL + path
	if params != "" {
		url = url + "?" + params
	}
	req, err := c.createHTTPRequest(method, url, body)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, kebError.NewTemporaryError(err.Error())
	}
	defer func() {
		// go ahead (close body) even if draining fails
		_ = iosafety.DrainReader(resp.Body)
		_ = resp.Body.Close()
	}()

	switch {
	case resp.StatusCode < 300:
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return resp.StatusCode, kebError.NewTemporaryError("unable to read response body")
		}
		err = json.Unmarshal(body, response)
		if err != nil {
			return resp.StatusCode, errors.Wrapf(err, "while unmarshalling response: %s", string(body))
		}

		return resp.StatusCode, nil
	case resp.StatusCode > 500:
		return resp.StatusCode, kebError.NewTemporaryError("unable to call %s, got status %d", url, resp.StatusCode)
	default:
		msg := ""
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			msg = string(body)
		}
		return resp.StatusCode, fmt.Errorf("error when calling url %s, got status %d: %s", url, resp.StatusCode, msg)
	}
}

func (c *client) get(path string, response interface{}, q *query.Parameters) error {
	_, err := c.call(http.MethodGet, path, response, q, nil)
	return err
}
