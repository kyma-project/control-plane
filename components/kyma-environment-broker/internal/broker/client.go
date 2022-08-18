package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"

	log "github.com/sirupsen/logrus"
)

const (
	kymaClassID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"

	instancesURL       = "/oauth/v2/service_instances"
	deprovisionTmpl    = "%s%s/%s?service_id=%s&plan_id=%s"
	updateInstanceTmpl = "%s%s/%s"
)

type (
	contextDTO struct {
		GlobalAccountID string `json:"globalaccount_id"`
		SubAccountID    string `json:"subaccount_id"`
		Active          *bool  `json:"active"`
	}

	parametersDTO struct {
		Expired *bool `json:"expired"`
	}

	serviceUpdatePatchDTO struct {
		ServiceID  string        `json:"service_id"`
		PlanID     string        `json:"plan_id"`
		Context    contextDTO    `json:"context"`
		Parameters parametersDTO `json:"parameters"`
	}

	serviceInstancesResponseDTO struct {
		Operation string `json:"operation"`
	}

	errorResponse struct {
		Error       string `json:"error"`
		Description string `json:"description"`
	}
)

type ClientConfig struct {
	URL          string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
}

type Client struct {
	brokerConfig ClientConfig
	httpClient   *http.Client
}

func NewClient(ctx context.Context, config ClientConfig) *Client {
	cfg := clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{config.Scope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	return &Client{
		brokerConfig: config,
		httpClient:   httpClientOAuth,
	}
}

// Deprovision requests Runtime deprovisioning in KEB with given details
func (c *Client) Deprovision(instance internal.Instance) (string, error) {
	deprovisionURL, err := c.formatDeprovisionUrl(instance)
	if err != nil {
		return "", err
	}

	response := serviceInstancesResponseDTO{}
	log.Infof("Requesting deprovisioning of the environment with instance id: %q", instance.InstanceID)
	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequestWithPoll(http.MethodDelete, deprovisionURL, http.StatusAccepted, nil, &response)
		if err != nil {
			log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "while waiting for successful deprovision call")
	}
	return response.Operation, nil
}

// SendExpirationRequest requests Runtime suspension due to expiration
func (c *Client) SendExpirationRequest(instance internal.Instance) (string, error) {
	request, err := preparePatchRequest(instance, c.brokerConfig.URL)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return "", errors.Wrapf(err, "while executing request URL: %s", request.URL)
	}
	defer c.warnOnError(resp.Body.Close)

	return processResponse(resp.StatusCode, resp)
}

func processResponse(statusCode int, resp *http.Response) (string, error) {
	switch statusCode {
	case http.StatusAccepted, http.StatusOK:
		{
			log.Infof("Request accepted with status: %+v", statusCode)
			operation, err := decodeOperation(resp)
			if err != nil {
				return "", err
			}
			log.Infof("operation: %+v", operation)
			return operation, nil
		}
	case http.StatusUnprocessableEntity:
		{
			log.Warnf("Entity unprocessable - request rejected with status: %+v", statusCode)
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return "", err
			}
			log.Warnf("error: %+v description: %+v", errorString, description)
			return "", nil
		}
	default:
		{
			if statusCode >= 200 && statusCode <= 299 {
				return "", fmt.Errorf("request with unexpected status: %+v", statusCode)
			}
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return "", err
			}
			return "", fmt.Errorf("error: %+v description: %+v", errorString, description)
		}
	}
}

func decodeOperation(resp *http.Response) (string, error) {
	response := serviceInstancesResponseDTO{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", errors.Wrapf(err, "while decoding response body")
	}
	return response.Operation, nil
}

func decodeErrorResponse(resp *http.Response) (string, string, error) {
	response := errorResponse{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", "", errors.Wrapf(err, "while decoding error response body")
	}
	return response.Description, response.Error, nil
}

func preparePatchRequest(instance internal.Instance, brokerConfigURL string) (*http.Request, error) {
	updateInstanceUrl := fmt.Sprintf(updateInstanceTmpl, brokerConfigURL, instancesURL, instance.InstanceID)

	jsonPayload, err := preparePayload(instance)
	if err != nil {
		return nil, errors.Wrap(err, "while marshaling payload")
	}

	log.Infof("Requesting expiration of the environment with instance id: %q", instance.InstanceID)

	request, err := http.NewRequest(http.MethodPatch, updateInstanceUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(err, "while creating request for Kyma Environment Broker")
	}
	request.Header.Set("X-Broker-API-Version", "2.14")
	return request, nil
}

func preparePayload(instance internal.Instance) ([]byte, error) {
	expired := true
	active := false
	payload := serviceUpdatePatchDTO{
		ServiceID: KymaServiceID,
		PlanID:    instance.ServicePlanID,
		Context: contextDTO{
			GlobalAccountID: instance.SubscriptionGlobalAccountID,
			SubAccountID:    instance.SubAccountID,
			Active:          &active},
		Parameters: parametersDTO{Expired: &expired}}
	jsonPayload, err := json.Marshal(payload)
	return jsonPayload, err
}

func (c *Client) formatDeprovisionUrl(instance internal.Instance) (string, error) {
	if len(instance.ServicePlanID) == 0 {
		return "", errors.Errorf("empty ServicePlanID")
	}

	return fmt.Sprintf(deprovisionTmpl, c.brokerConfig.URL, instancesURL, instance.InstanceID, kymaClassID, instance.ServicePlanID), nil
}

func (c *Client) executeRequestWithPoll(method, url string, expectedStatus int, body io.Reader, responseBody interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return errors.Wrap(err, "while creating request for provisioning")
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "while executing request URL: %s", url)
	}
	defer c.warnOnError(resp.Body.Close)
	if resp.StatusCode != expectedStatus {
		return errors.Errorf("got unexpected status code while calling Kyma Environment Broker: want: %d, got: %d", expectedStatus, resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return errors.Wrapf(err, "while decoding body")
	}

	return nil
}

func (c *Client) warnOnError(do func() error) {
	if err := do(); err != nil {
		log.Warn(err.Error())
	}
}

// setHttpClient auxiliary method of testing to get rid of oAuth client wrapper
func (c *Client) setHttpClient(httpClient *http.Client) {
	c.httpClient = httpClient
}
