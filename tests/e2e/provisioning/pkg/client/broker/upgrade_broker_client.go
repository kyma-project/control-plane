package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	infoRuntimePath      = "%s/info/runtimes"
	upgradeInstancePath  = "%s/upgrade/kyma"
	getOrchestrationPath = "%s/orchestrations/%s"
)

type UpgradeClient struct {
	log    logrus.FieldLogger
	URL    string
	client *http.Client
}

func NewUpgradeClient(ctx context.Context, oAuthConfig BrokerOAuthConfig, config Config, log logrus.FieldLogger) *UpgradeClient {
	cfg := clientcredentials.Config{
		ClientID:     oAuthConfig.ClientID,
		ClientSecret: oAuthConfig.ClientSecret,
		TokenURL:     config.TokenURL,
		Scopes:       []string{oAuthConfig.Scope},
	}
	httpClientOAuth := cfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second

	return &UpgradeClient{
		log:    log.WithField("client", "upgrade_broker_client"),
		URL:    config.URL,
		client: httpClientOAuth,
	}
}

func (c *UpgradeClient) UpgradeRuntime(runtimeID string) (string, error) {
	payload := UpgradeRuntimeRequest{
		Targets: Target{
			Include: []RuntimeTarget{{RuntimeID: runtimeID}},
		},
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Wrap(err, "while marshaling payload request")
	}

	response, err := c.executeRequest(http.MethodPost, fmt.Sprintf(upgradeInstancePath, c.URL), bytes.NewReader(requestBody))
	if err != nil {
		return "", errors.Wrap(err, "while executing upgrade runtime request")
	}
	if response.StatusCode != http.StatusAccepted {
		return "", c.handleUnsupportedStatusCode(response)
	}

	upgradeResponse := &UpgradeRuntimeResponse{}
	err = json.NewDecoder(response.Body).Decode(upgradeResponse)
	if err != nil {
		return "", errors.Wrap(err, "while decoding upgrade response")
	}

	return upgradeResponse.OrchestrationID, nil
}

func (c *UpgradeClient) FetchRuntimeID(instanceID string) (string, error) {
	var runtimeID string
	err := wait.Poll(3*time.Second, 1*time.Minute, func() (bool, error) {
		id, permanentError, err := c.fetchRuntimeID(instanceID)
		if err != nil && permanentError {
			return true, errors.Wrap(err, "cannot fetch runtimeID")
		}
		if err != nil {
			c.log.Warnf("runtime is not ready: %s ...", err)
			return false, nil
		}
		runtimeID = id
		return true, nil
	})
	if err != nil {
		return runtimeID, errors.Wrap(err, "while waiting for runtimeID")
	}

	return runtimeID, nil
}

func (c *UpgradeClient) fetchRuntimeID(instanceID string) (string, bool, error) {
	response, err := c.executeRequest(http.MethodGet, fmt.Sprintf(infoRuntimePath, c.URL), nil)
	if err != nil {
		return "", false, errors.Wrap(err, "while executing fetch runtime request")
	}
	if response.StatusCode != http.StatusOK {
		return "", false, c.handleUnsupportedStatusCode(response)
	}

	var runtimes []Runtime
	err = json.NewDecoder(response.Body).Decode(&runtimes)
	if err != nil {
		return "", true, errors.Wrap(err, "while decoding upgrade response")
	}

	for _, runtime := range runtimes {
		if runtime.ServiceInstanceID != instanceID {
			continue
		}
		if runtime.RuntimeID == "" {
			continue
		}
		return runtime.RuntimeID, false, nil
	}

	return "", false, errors.Errorf("runtimeID for instanceID %s not exist", instanceID)
}

func (c *UpgradeClient) AwaitOperationFinished(orchestrationID string, timeout time.Duration) error {
	err := wait.Poll(10*time.Second, timeout, func() (bool, error) {
		permanentError, err := c.awaitOperationFinished(orchestrationID)
		if err != nil && permanentError {
			return true, errors.Wrap(err, "cannot fetch operation status")
		}
		if err != nil {
			c.log.Warnf("upgrade is not ready: %s ...", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for upgrade operation finished")
	}

	return nil
}

func (c *UpgradeClient) awaitOperationFinished(orchestrationID string) (bool, error) {
	response, err := c.executeRequest(http.MethodGet, fmt.Sprintf(getOrchestrationPath, c.URL, orchestrationID), nil)
	if err != nil {
		return false, errors.Wrap(err, "while executing get orchestration request")
	}

	if response.StatusCode != http.StatusOK {
		return false, c.handleUnsupportedStatusCode(response)
	}

	orchestrationResponse := &OrchestrationResponse{}
	err = json.NewDecoder(response.Body).Decode(orchestrationResponse)
	if err != nil {
		return true, errors.Wrap(err, "while decoding orchestration response")
	}

	switch orchestrationResponse.State {
	case "succeeded":
		return false, nil
	case "failed":
		return true, errors.Errorf("operation is in failed state")
	default:
		return false, errors.Errorf("operation is in %s state", orchestrationResponse.State)
	}
}

func (c *UpgradeClient) executeRequest(method, url string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return &http.Response{}, errors.Wrap(err, "while creating request for KEB")
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	response, err := c.client.Do(request)
	if err != nil {
		return &http.Response{}, errors.Wrapf(err, "while executing request to KEB on: %s", url)
	}

	return response, nil
}

func (c *UpgradeClient) handleUnsupportedStatusCode(response *http.Response) error {
	var body string
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		body = "cannot read body response"
	} else {
		body = string(responseBody)
	}

	return errors.Wrapf(err, "unsupported status code %d: (%s)", response.StatusCode, body)
}
