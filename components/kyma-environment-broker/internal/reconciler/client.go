package reconciler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/sirupsen/logrus"
)

//go:generate mockery -name=Client -output=automock -outpkg=automock -case=underscore

type Client interface {
	ApplyClusterConfig(cluster Cluster) (*State, error)
	DeleteCluster(clusterName string) error
	GetCluster(clusterName string, configVersion int64) (*State, error)
	GetLatestCluster(clusterName string) (*State, error)
	GetStatusChange(clusterName, offset string) ([]*StatusChange, error)
}

type Config struct {
	URL         string
	DumpRequest bool `envconfig:"default=false"`
}

type client struct {
	httpClient *http.Client
	log        logrus.FieldLogger
	config     *Config
}

func NewReconcilerClient(httpClient *http.Client, log logrus.FieldLogger, cfg *Config) *client {
	return &client{
		httpClient: httpClient,
		log:        log,
		config:     cfg,
	}
}

// POST /v1/clusters
func (c *client) ApplyClusterConfig(cluster Cluster) (*State, error) {
	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	reader := bytes.NewReader(reqBody)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/clusters", c.config.URL), reader)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	if c.config.DumpRequest {
		c.log.Debugf(string(reqBody))
	}
	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, kebError.NewTemporaryError(err.Error())
	}
	defer res.Body.Close()

	c.log.Debugf("Got response: statusCode=%d", res.StatusCode)
	switch {
	case res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated:
	case res.StatusCode >= 400 && res.StatusCode < 500:
		return nil, fmt.Errorf("got status %d", res.StatusCode)
	case res.StatusCode >= 500:
		return nil, kebError.NewTemporaryError("Got status %d", res.StatusCode)
	}

	registerClusterResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	var response *State
	err = json.Unmarshal(registerClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	return response, nil
}

// DELETE /v1/clusters/{clusterName}
func (c *client) DeleteCluster(clusterName string) error {
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/clusters/%s", c.config.URL, clusterName), nil)
	if err != nil {
		c.log.Error(err)
		return err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return kebError.NewTemporaryError(err.Error())
	}
	switch {
	case res.StatusCode == http.StatusNotFound:
		return nil
	case res.StatusCode >= 400 && res.StatusCode < 500 && res.StatusCode != http.StatusNotFound:
		return fmt.Errorf("got status %d", res.StatusCode)
	case res.StatusCode >= 500:
		return kebError.NewTemporaryError("Got status %d", res.StatusCode)
	default:
		return nil
	}

}

// GET /v1/clusters/{clusterName}/configs/{configVersion}/status
func (c *client) GetCluster(clusterName string, configVersion int64) (*State, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/configs/%d/status", c.config.URL, clusterName, configVersion), nil)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, kebError.NewTemporaryError(err.Error())
	}
	defer res.Body.Close()

	getClusterResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	var response *State
	err = json.Unmarshal(getClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	return response, nil
}

// GET v1/clusters/{clusterName}/status
func (c *client) GetLatestCluster(clusterName string) (*State, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/status", c.config.URL, clusterName), nil)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, kebError.NewTemporaryError(err.Error())
	}
	defer res.Body.Close()

	getClusterResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	var response *State
	err = json.Unmarshal(getClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	return response, nil
}

// GET v1/clusters/{clusterName}/statusChanges/{offset}
// offset is parsed to time.Duration
func (c *client) GetStatusChange(clusterName, offset string) ([]*StatusChange, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/statusChanges/%s", c.config.URL, clusterName, offset), nil)
	if err != nil {
		c.log.Error(err)
		return []*StatusChange{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return []*StatusChange{}, err
	}
	defer res.Body.Close()

	getStatusChangeResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return []*StatusChange{}, err
	}
	var response []*StatusChange
	err = json.Unmarshal(getStatusChangeResponse, &response)
	if err != nil {
		c.log.Error(err)
		return []*StatusChange{}, err
	}
	return response, nil
}
