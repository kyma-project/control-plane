package reconciler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

//go:generate mockery -name=Client -output=automock -outpkg=automock -case=underscore

type Client interface {
	RegisterCluster(cluster Cluster) (*State, error)
	UpdateCluster(cluster Cluster) (*State, error)
	DeleteCluster(clusterName string) error
	GetCluster(clusterName, configVersion string) (*State, error)
	GetLatestCluster(clusterName string) (*State, error)
	GetStatusChange(clusterName, offset string) ([]*StatusChange, error)
}

type Config struct {
	reconcilerURL string
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
func (c *client) RegisterCluster(cluster Cluster) (*State, error) {
	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	reader := bytes.NewReader(reqBody)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/clusters", c.config.reconcilerURL), reader)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	defer res.Body.Close()
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

// PUT /v1/clusters
func (c *client) UpdateCluster(cluster Cluster) (*State, error) {
	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	reader := bytes.NewReader(reqBody)

	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/clusters", c.config.reconcilerURL), reader)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}
	defer res.Body.Close()

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
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/clusters/%s", c.config.reconcilerURL, clusterName), nil)
	if err != nil {
		c.log.Error(err)
		return err
	}

	_, err = c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return err
	}

	return nil
}

// GET /v1/clusters/{clusterName}/configs/{configVersion}/status
func (c *client) GetCluster(clusterName, configVersion string) (*State, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/configs/%s/status", c.config.reconcilerURL, clusterName, configVersion), nil)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
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
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/status", c.config.reconcilerURL, clusterName), nil)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
	}

	res, err := c.httpClient.Do(request)
	if err != nil {
		c.log.Error(err)
		return &State{}, err
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
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/clusters/%s/statusChanges/%s", c.config.reconcilerURL, clusterName, offset), nil)
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
