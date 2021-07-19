package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-incubator/compass/tests/director/pkg/ptr"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/thanhpk/randstr"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Config struct {
	ClientName string
	TokenURL   string
	URL        string
	PlanID     string
	Region     string `envconfig:"optional"`
}

type BrokerOAuthConfig struct {
	ClientID     string
	ClientSecret string
	Scope        string
}

type Client struct {
	brokerConfig    Config
	clusterName     string
	instanceID      string
	globalAccountID string
	subAccountID    string
	userID          string

	client *http.Client
	log    logrus.FieldLogger
}

func NewClient(ctx context.Context, config Config, globalAccountID, instanceID, subAccountID, userID string, oAuthCfg BrokerOAuthConfig, log logrus.FieldLogger) *Client {
	cfg := clientcredentials.Config{
		ClientID:     oAuthCfg.ClientID,
		ClientSecret: oAuthCfg.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{oAuthCfg.Scope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	return &Client{
		brokerConfig:    config,
		instanceID:      instanceID,
		clusterName:     fmt.Sprintf("%s-%s", "e2e-provisioning", strings.ToLower(randstr.String(10))),
		globalAccountID: globalAccountID,
		client:          httpClientOAuth,
		log:             log,
		subAccountID:    subAccountID,
		userID:          userID,
	}
}

const (
	kymaClassID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
)

type inputContext struct {
	TenantID        string `json:"tenant_id"`
	SubAccountID    string `json:"subaccount_id"`
	UserID          string `json:"user_id"`
	GlobalAccountID string `json:"globalaccount_id"`
	Active          *bool  `json:"active,omitempty"`
}

type provisionResponse struct {
	Operation string `json:"operation"`
}

type lastOperationResponse struct {
	State string `json:"state"`
}

type instanceDetailsResponse struct {
	DashboardURL string `json:"dashboard_url"`
}

type provisionParameters struct {
	Name        string   `json:"name"`
	Components  []string `json:"components"`
	KymaVersion string   `json:"kymaVersion,omitempty"`
}

// ProvisionRuntime requests Runtime provisioning in KEB
// kymaVersion is optional, if it is empty, the default KEB version will be used
func (c *Client) ProvisionRuntime(kymaVersion string) (string, error) {
	c.log.Infof("Provisioning Runtime [instanceID: %s, NAME: %s]", c.InstanceID(), c.ClusterName())
	requestByte, err := c.prepareProvisionDetails(kymaVersion)
	if err != nil {
		return "", errors.Wrap(err, "while preparing provision details")
	}
	c.log.Infof("Provisioning parameters: %v", string(requestByte))

	provisionURL := fmt.Sprintf("%s/service_instances/%s", c.baseURL(), c.InstanceID())
	response := provisionResponse{}
	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequest(http.MethodPut, provisionURL, http.StatusAccepted, bytes.NewReader(requestByte), &response)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		if response.Operation == "" {
			c.log.Warn("Got empty operation ID")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "while waiting for successful provision call")
	}
	c.log.Infof("Successfully send provision request, got operation ID %s", response.Operation)

	return response.Operation, nil
}

func (c *Client) DeprovisionRuntime() (string, error) {
	format := "%s/service_instances/%s?service_id=%s&plan_id=%s"
	deprovisionURL := fmt.Sprintf(format, c.baseURL(), c.InstanceID(), kymaClassID, c.brokerConfig.PlanID)

	response := provisionResponse{}
	c.log.Infof("Deprovisioning Runtime [ID: %s, NAME: %s]", c.instanceID, c.clusterName)
	err := wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequest(http.MethodDelete, deprovisionURL, http.StatusAccepted, nil, &response)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "while waiting for successful deprovision call")
	}
	c.log.Infof("Successfully send deprovision request, got operation ID %s", response.Operation)
	return response.Operation, nil
}

func (c *Client) SuspendRuntime() error {
	c.log.Infof("Suspending Runtime [instanceID: %s, NAME: %s]", c.instanceID, c.clusterName)
	requestByte, err := c.prepareUpdateDetails(ptr.Bool(false))
	if err != nil {
		return errors.Wrap(err, "while preparing update details")
	}
	c.log.Infof("Suspension parameters: %v", string(requestByte))

	format := "%s/service_instances/%s"
	suspensionURL := fmt.Sprintf(format, c.baseURL(), c.instanceID)

	suspensionResponse := instanceDetailsResponse{}
	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequest(http.MethodPatch, suspensionURL, http.StatusOK, bytes.NewReader(requestByte), &suspensionResponse)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for successful suspension call")
	}
	c.log.Infof("Successfully send suspension request for %s", suspensionResponse)
	return nil
}

func (c *Client) UnsuspendRuntime() error {
	c.log.Infof("Unsuspending Runtime [instanceID: %s, NAME: %s]", c.InstanceID(), c.ClusterName())
	requestByte, err := c.prepareUpdateDetails(ptr.Bool(true))
	if err != nil {
		return errors.Wrap(err, "while preparing update details")
	}
	c.log.Infof("Unuspension parameters: %v", string(requestByte))

	format := "%s/service_instances/%s"
	suspensionURL := fmt.Sprintf(format, c.baseURL(), c.InstanceID())

	unsuspensionResponse := instanceDetailsResponse{}
	err = wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequest(http.MethodPatch, suspensionURL, http.StatusOK, bytes.NewReader(requestByte), &unsuspensionResponse)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for successful unsuspension call")
	}
	c.log.Infof("Successfully send unsuspension request for %s", unsuspensionResponse)
	return nil
}

func (c *Client) GlobalAccountID() string {
	return c.globalAccountID
}

func (c *Client) InstanceID() string {
	return c.instanceID
}

func (c *Client) SetInstanceID(id string) {
	c.instanceID = id
}

func (c *Client) SubAccountID() string {
	return c.subAccountID
}

func (c *Client) UserID() string {
	return c.userID
}

func (c *Client) ClusterName() string {
	return c.clusterName
}

func (c *Client) AwaitOperationSucceeded(operationID string, timeout time.Duration) error {
	lastOperationURL := fmt.Sprintf("%s/service_instances/%s/last_operation?operation=%s", c.baseURL(), c.InstanceID(), operationID)
	if operationID == "" {
		lastOperationURL = fmt.Sprintf("%s/service_instances/%s/last_operation", c.baseURL(), c.InstanceID())
	}

	c.log.Infof("Waiting for operation at most %s", timeout.String())

	response := lastOperationResponse{}
	err := wait.Poll(5*time.Minute, timeout, func() (bool, error) {
		err := c.executeRequest(http.MethodGet, lastOperationURL, http.StatusOK, nil, &response)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		c.log.Infof("Last operation status: %s", response.State)
		switch domain.LastOperationState(response.State) {
		case domain.Succeeded:
			c.log.Infof("Operation succeeded!")
			return true, nil
		case domain.InProgress:
			return false, nil
		case domain.Failed:
			c.log.Info("Operation failed!")
			return true, errors.New("provisioning failed")
		default:
			if response.State == "" {
				c.log.Infof("Got empty last operation response")
				return false, nil
			}
			return false, nil
		}
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for succeeded last operation")
	}
	return nil
}

func (c *Client) FetchDashboardURL() (string, error) {
	instanceDetailsURL := fmt.Sprintf("%s/service_instances/%s", c.baseURL(), c.InstanceID())

	c.log.Info("Fetching the Runtime's dashboard URL")
	response := instanceDetailsResponse{}
	err := wait.Poll(time.Second, time.Second*5, func() (bool, error) {
		err := c.executeRequest(http.MethodGet, instanceDetailsURL, http.StatusOK, nil, &response)
		if err != nil {
			c.log.Warn(errors.Wrap(err, "while executing request").Error())
			return false, nil
		}
		if response.DashboardURL == "" {
			c.log.Warn("got empty dashboardURL")
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", errors.Wrap(err, "while waiting for dashboardURL")
	}
	c.log.Infof("Successfully fetched dashboard URL: %s", response.DashboardURL)

	return response.DashboardURL, nil
}

func (c *Client) prepareProvisionDetails(customVersion string) ([]byte, error) {
	parameters := provisionParameters{
		Name:        c.clusterName,
		Components:  []string{},    // fill with optional components
		KymaVersion: customVersion, // If empty filed will be omitted
	}
	ctx := inputContext{
		TenantID:        "1eba80dd-8ff6-54ee-be4d-77944d17b10b",
		SubAccountID:    c.subAccountID,
		UserID:          c.userID,
		GlobalAccountID: c.globalAccountID,
	}
	rawParameters, err := json.Marshal(parameters)
	if err != nil {
		return nil, errors.Wrap(err, "while marshalling parameters body")
	}
	rawContext, err := json.Marshal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "while marshalling context body")
	}
	requestBody := domain.ProvisionDetails{
		ServiceID:        kymaClassID,
		PlanID:           c.brokerConfig.PlanID,
		OrganizationGUID: uuid.New().String(),
		SpaceGUID:        uuid.New().String(),
		RawContext:       rawContext,
		RawParameters:    rawParameters,
		MaintenanceInfo: &domain.MaintenanceInfo{
			Version:     "0.1.0",
			Description: "Kyma environment broker e2e-provisioning test",
		},
	}

	requestByte, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "while marshalling request body")
	}
	return requestByte, nil
}

func (c *Client) prepareUpdateDetails(active *bool) ([]byte, error) {
	ctx := inputContext{
		TenantID:        "1eba80dd-8ff6-54ee-be4d-77944d17b10b",
		SubAccountID:    c.subAccountID,
		GlobalAccountID: c.globalAccountID,
		Active:          active,
	}
	rawContext, err := json.Marshal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "while marshalling context body")
	}
	requestBody := domain.UpdateDetails{
		ServiceID:     kymaClassID,
		PlanID:        c.brokerConfig.PlanID,
		RawContext:    rawContext,
		MaintenanceInfo: &domain.MaintenanceInfo{
			Version:     "0.1.0",
			Description: "Kyma environment broker e2e-provisioning test",
		},
	}
	requestByte, err := json.Marshal(requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "while marshalling request body")
	}
	return requestByte, nil
}

func (c *Client) executeRequest(method, url string, expectedStatus int, body io.Reader, responseBody interface{}) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return errors.Wrap(err, "while creating request for provisioning")
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	resp, err := c.client.Do(request)
	if err != nil {
		return errors.Wrapf(err, "while executing request URL: %s", url)
	}
	defer c.warnOnError(resp.Body.Close)
	if resp.StatusCode != expectedStatus {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		c.log.Warnf("%s", bodyString)
		return errors.Errorf("got unexpected status code while calling Kyma Environment Broker: want: %d, got: %d (url=%s)", expectedStatus, resp.StatusCode, url)
	}

	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return errors.Wrapf(err, "while decoding body")
	}

	return nil
}

func (c *Client) warnOnError(do func() error) {
	if err := do(); err != nil {
		c.log.Warn(err.Error())
	}
}

func (c *Client) baseURL() string {
	base := fmt.Sprintf("%s/oauth", c.brokerConfig.URL)
	if c.brokerConfig.Region == "" {
		return fmt.Sprintf("%s/v2", base)
	}
	return fmt.Sprintf("%s/%s/v2", base, c.brokerConfig.Region)
}
