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
	"golang.org/x/oauth2/clientcredentials"

	log "github.com/sirupsen/logrus"
)

const (
	kymaClassID       = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	AccountCleanupJob = "accountcleanup-job"

	instancesURL       = "/oauth/v2/service_instances"
	deprovisionTmpl    = "%s%s/%s?service_id=%s&plan_id=%s"
	updateInstanceTmpl = "%s%s/%s"
	getInstanceTmpl    = "%s%s/%s"
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
	TokenURL     string `envconfig:"optional"`
	ClientID     string `envconfig:"optional"`
	ClientSecret string `envconfig:"optional"`
	Scope        string `envconfig:"optional"`
}

type Client struct {
	brokerConfig ClientConfig
	httpClient   *http.Client
	poller       Poller
	UserAgent    string
}

func NewClientConfig(URL string) *ClientConfig {
	return &ClientConfig{
		URL: URL,
	}
}

func NewClient(ctx context.Context, config ClientConfig) *Client {
	return NewClientWithPoller(ctx, config, NewDefaultPoller())
}

func NewClientWithPoller(ctx context.Context, config ClientConfig, poller Poller) *Client {
	if config.TokenURL == "" {
		return &Client{
			brokerConfig: config,
			httpClient:   http.DefaultClient,
			poller:       poller,
		}
	}
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
		poller:       poller,
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
	err = c.poller.Invoke(func() (bool, error) {
		err := c.executeRequestWithPoll(http.MethodDelete, deprovisionURL, http.StatusAccepted, nil, &response)
		if err != nil {
			log.Warn(fmt.Sprintf("while executing request: %s", err.Error()))
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return "", fmt.Errorf("while waiting for successful deprovision call: %w", err)
	}

	return response.Operation, nil
}

// SendExpirationRequest requests Runtime suspension due to expiration
func (c *Client) SendExpirationRequest(instance internal.Instance) (suspensionUnderWay bool, err error) {
	request, err := preparePatchRequest(instance, c.brokerConfig.URL)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("while executing request URL: %s for instanceID: %s: %w", request.URL,
			instance.InstanceID, err)
	}
	defer c.warnOnError(resp.Body.Close)

	return processResponse(instance.InstanceID, resp.StatusCode, resp)
}

func (c *Client) GetInstanceRequest(instanceID string) (response *http.Response, err error) {
	request, err := prepareGetRequest(instanceID, c.brokerConfig.URL)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("while executing request URL: %s for instanceID: %s: %w", request.URL,
			instanceID, err)
	}
	defer c.warnOnError(resp.Body.Close)

	return resp, nil
}

func processResponse(instanceID string, statusCode int, resp *http.Response) (suspensionUnderWay bool, err error) {
	switch statusCode {
	case http.StatusAccepted, http.StatusOK:
		{
			log.Infof("Request for instanceID: %s accepted with status: %+v", instanceID, statusCode)
			operation, err := decodeOperation(resp)
			if err != nil {
				return false, err
			}
			log.Infof("For instanceID: %s we received operation: %s", instanceID, operation)
			return true, nil
		}
	case http.StatusUnprocessableEntity:
		{
			log.Warnf("For instanceID: %s we received entity unprocessable - status: %+v", instanceID, statusCode)
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return false, fmt.Errorf("for instanceID: %s: %w", instanceID, err)
			}
			log.Warnf("error: %+v description: %+v instanceID: %s", errorString, description, instanceID)
			return false, nil
		}
	default:
		{
			if statusCode >= 200 && statusCode <= 299 {
				return false, fmt.Errorf("for instanceID: %s we received unexpected status: %+v", instanceID, statusCode)
			}
			description, errorString, err := decodeErrorResponse(resp)
			if err != nil {
				return false, fmt.Errorf("for instanceID: %s: %w", instanceID, err)
			}
			return false, fmt.Errorf("error: %+v description: %+v instanceID: %s", errorString, description, instanceID)
		}
	}
}

func decodeOperation(resp *http.Response) (string, error) {
	response := serviceInstancesResponseDTO{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("while decoding response body: %w", err)
	}
	return response.Operation, nil
}

func decodeErrorResponse(resp *http.Response) (string, string, error) {
	response := errorResponse{}
	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", "", fmt.Errorf("while decoding error response body: %w", err)
	}
	return response.Description, response.Error, nil
}

func preparePatchRequest(instance internal.Instance, brokerConfigURL string) (*http.Request, error) {
	updateInstanceUrl := fmt.Sprintf(updateInstanceTmpl, brokerConfigURL, instancesURL, instance.InstanceID)

	jsonPayload, err := preparePayload(instance)
	if err != nil {
		return nil, fmt.Errorf("while marshaling payload for instanceID: %s: %w", instance.InstanceID, err)
	}

	log.Infof("Requesting expiration of the environment with instance id: %q", instance.InstanceID)

	request, err := http.NewRequest(http.MethodPatch, updateInstanceUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("while creating request for instanceID: %s: %w", instance.InstanceID, err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")
	return request, nil
}

func prepareGetRequest(instanceID string, brokerConfigURL string) (*http.Request, error) {
	getInstanceUrl := fmt.Sprintf(getInstanceTmpl, brokerConfigURL, instancesURL, instanceID)

	request, err := http.NewRequest(http.MethodGet, getInstanceUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("while creating GET request for instanceID: %s: %w", instanceID, err)
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
		return "", fmt.Errorf("empty ServicePlanID")
	}

	return fmt.Sprintf(deprovisionTmpl, c.brokerConfig.URL, instancesURL, instance.InstanceID, kymaClassID, instance.ServicePlanID), nil
}

func (c *Client) executeRequestWithPoll(method, url string, expectedStatus int, body io.Reader, responseBody interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("while creating request for provisioning: %w", err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")
	if len(c.UserAgent) != 0 {
		request.Header.Set("User-Agent", c.UserAgent)
	}

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("while executing request URL: %s: %w", url, err)
	}
	defer c.warnOnError(resp.Body.Close)
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("got unexpected status code while calling Kyma Environment Broker: want: %d, got: %d",
			expectedStatus, resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return fmt.Errorf("while decoding body: %w", err)
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
