package reconciler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"

	"github.com/sirupsen/logrus"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
)

//go:generate mockery -name=Client -output=automock -outpkg=automock -case=underscore

type Client interface {
	ApplyClusterConfig(cluster reconcilerApi.Cluster) (*reconcilerApi.HTTPClusterResponse, error)
	DeleteCluster(clusterName string) error
	GetCluster(clusterName string, configVersion int64) (*reconcilerApi.HTTPClusterResponse, error)
	GetLatestCluster(clusterName string) (*reconcilerApi.HTTPClusterResponse, error)
	GetStatusChange(clusterName, offset string) ([]*reconcilerApi.StatusChange, error)
}

type Config struct {
	URL string
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
func (c *client) ApplyClusterConfig(cluster reconcilerApi.Cluster) (*reconcilerApi.HTTPClusterResponse, error) {
	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}

	reader := bytes.NewReader(reqBody)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/clusters", c.config.URL), reader)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, kebError.NewTemporaryError(err.Error())
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
		return &reconcilerApi.HTTPClusterResponse{}, err
	}
	var response *reconcilerApi.HTTPClusterResponse
	err = json.Unmarshal(registerClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
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
func (c *client) GetCluster(clusterName string, configVersion int64) (*reconcilerApi.HTTPClusterResponse, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/configs/%d/status", c.config.URL, clusterName, configVersion), nil)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, kebError.NewTemporaryError(err.Error())
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return &reconcilerApi.HTTPClusterResponse{}, kebError.NotFoundError{}
	case res.StatusCode >= 400 && res.StatusCode < 500 && res.StatusCode != http.StatusNotFound:
		return &reconcilerApi.HTTPClusterResponse{}, fmt.Errorf("got status %d", res.StatusCode)
	case res.StatusCode >= 500:
		return &reconcilerApi.HTTPClusterResponse{}, kebError.NewTemporaryError("Got status %d", res.StatusCode)
	}

	getClusterResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}
	var response *reconcilerApi.HTTPClusterResponse
	err = json.Unmarshal(getClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}
	return response, nil
}

// GET v1/clusters/{clusterName}/status
func (c *client) GetLatestCluster(clusterName string) (*reconcilerApi.HTTPClusterResponse, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/status", c.config.URL, clusterName), nil)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, kebError.NewTemporaryError(err.Error())
	}
	defer res.Body.Close()

	getClusterResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}
	var response *reconcilerApi.HTTPClusterResponse
	err = json.Unmarshal(getClusterResponse, &response)
	if err != nil {
		c.log.Error(err)
		return &reconcilerApi.HTTPClusterResponse{}, err
	}
	return response, nil
}

// GET v1/clusters/{clusterName}/statusChanges/{offset}
// offset is parsed to time.Duration
func (c *client) GetStatusChange(clusterName, offset string) ([]*reconcilerApi.StatusChange, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/statusChanges/%s", c.config.URL, clusterName, offset), nil)
	if err != nil {
		c.log.Error(err)
		return []*reconcilerApi.StatusChange{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return []*reconcilerApi.StatusChange{}, err
	}
	defer res.Body.Close()

	getStatusChangeResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		c.log.Error(err)
		return []*reconcilerApi.StatusChange{}, err
	}
	var response []*reconcilerApi.StatusChange
	err = json.Unmarshal(getStatusChangeResponse, &response)
	if err != nil {
		c.log.Error(err)
		return []*reconcilerApi.StatusChange{}, err
	}
	return response, nil
}
