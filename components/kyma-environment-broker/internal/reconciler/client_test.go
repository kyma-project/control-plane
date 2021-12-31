package reconciler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var clusterJSONFile = filepath.Join(".", "test", "cluster.json")

const fixReconcilerURL = "reconciler-url:8080"

func Test_RegisterCluster(t *testing.T) {
	// given
	fixClusterID := "1"
	fixClusterVersion := int64(1)
	fixConfigVersion := int64(1)
	requestedCluster := fixCluster(t, fixClusterID, fixClusterVersion)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assert.Equal(t, "/v1/clusters", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		err := json.NewEncoder(w).Encode(reconcilerApi.HTTPClusterResponse{
			Cluster:              requestedCluster.RuntimeID,
			ClusterVersion:       fixClusterVersion,
			ConfigurationVersion: fixConfigVersion,
			Status:               reconcilerApi.StatusReconcilePending,
			StatusURL:            fmt.Sprintf("%s/v1/clusters/%s/configs/%s/status", fixReconcilerURL, requestedCluster.RuntimeID, strconv.FormatInt(fixConfigVersion, 10)),
		})
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewReconcilerClient(http.DefaultClient, logrus.New().WithField("client", "reconciler"), &Config{URL: ts.URL})

	// when
	response, err := client.ApplyClusterConfig(*requestedCluster)

	// then
	require.NoError(t, err)
	assert.Equal(t, requestedCluster.RuntimeID, response.Cluster)
	assert.Equal(t, fixClusterVersion, response.ClusterVersion)
	assert.Equal(t, fixConfigVersion, response.ConfigurationVersion)
	assert.Equal(t, reconcilerApi.StatusReconcilePending, response.Status)
	assert.Equal(t, fmt.Sprintf("%s/v1/clusters/%s/configs/%d/status", fixReconcilerURL, fixClusterID, fixConfigVersion), response.StatusURL)
}

func Test_DeleteCluster(t *testing.T) {
	// given
	fixClusterID := "1"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assert.Equal(t, fmt.Sprintf("/v1/clusters/%s", fixClusterID), r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		err := json.NewEncoder(w).Encode("")
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewReconcilerClient(http.DefaultClient, logrus.New().WithField("client", "reconciler"), &Config{URL: ts.URL})

	// when
	err := client.DeleteCluster(fixClusterID)

	// then
	require.NoError(t, err)
}

func Test_GetCluster(t *testing.T) {
	// given
	fixClusterID := "1"
	fixClusterVersion := int64(1)
	fixConfigVersion := int64(2)
	requestedCluster := fixCluster(t, fixClusterID, fixClusterVersion)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assert.Equal(t, fmt.Sprintf("/v1/clusters/%s/configs/%d/status", fixClusterID, fixConfigVersion), r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		err := json.NewEncoder(w).Encode(reconcilerApi.HTTPClusterResponse{
			Cluster:              requestedCluster.RuntimeID,
			ClusterVersion:       fixClusterVersion,
			ConfigurationVersion: fixConfigVersion,
			Status:               reconcilerApi.StatusReconcilePending,
		})
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewReconcilerClient(http.DefaultClient, logrus.New().WithField("client", "reconciler"), &Config{URL: ts.URL})

	// when
	response, err := client.GetCluster(fixClusterID, fixConfigVersion)

	// then
	require.NoError(t, err)
	assert.Equal(t, requestedCluster.RuntimeID, response.Cluster)
	assert.Equal(t, fixClusterVersion, response.ClusterVersion)
	assert.Equal(t, fixConfigVersion, response.ConfigurationVersion)
	assert.Equal(t, reconcilerApi.StatusReconcilePending, response.Status)
}

func Test_GetLatestCluster(t *testing.T) {
	// given
	fixClusterID := "1"
	fixClusterVersion := int64(1)
	fixConfigVersion := int64(2)
	requestedCluster := fixCluster(t, fixClusterID, fixClusterVersion)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assert.Equal(t, fmt.Sprintf("/v1/clusters/%s/status", fixClusterID), r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		err := json.NewEncoder(w).Encode(reconcilerApi.HTTPClusterResponse{
			Cluster:              requestedCluster.RuntimeID,
			ClusterVersion:       fixClusterVersion,
			ConfigurationVersion: fixConfigVersion,
			Status:               reconcilerApi.StatusReconcilePending,
		})
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewReconcilerClient(http.DefaultClient, logrus.New().WithField("client", "reconciler"), &Config{URL: ts.URL})

	// when
	response, err := client.GetLatestCluster(fixClusterID)

	// then
	require.NoError(t, err)
	assert.Equal(t, requestedCluster.RuntimeID, response.Cluster)
	assert.Equal(t, fixClusterVersion, response.ClusterVersion)
	assert.Equal(t, fixConfigVersion, response.ConfigurationVersion)
	assert.Equal(t, reconcilerApi.StatusReconcilePending, response.Status)
}

func Test_GetStatusChange(t *testing.T) {
	// given
	fixClusterID := "1"
	fixOffset := "1h"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assert.Equal(t, fmt.Sprintf("/v1/clusters/%s/statusChanges/%s", fixClusterID, fixOffset), r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		err := json.NewEncoder(w).Encode([]*reconcilerApi.StatusChange{
			{
				Status:   reconcilerApi.StatusReady,
				Duration: int64(40 * time.Second),
			},
			{
				Status:   reconcilerApi.StatusReconciling,
				Duration: int64(10 * time.Second),
			},
			{
				Status:   reconcilerApi.StatusReconcilePending,
				Duration: int64(30 * time.Second),
			},
		})
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := NewReconcilerClient(http.DefaultClient, logrus.New().WithField("client", "reconciler"), &Config{URL: ts.URL})

	// when
	response, err := client.GetStatusChange(fixClusterID, fixOffset)

	// then
	require.NoError(t, err)
	assert.Len(t, response, 3)
}

func fixCluster(t *testing.T, runtimeID string, clusterVersion int64) *reconcilerApi.Cluster {
	cluster := &reconcilerApi.Cluster{}
	data, err := ioutil.ReadFile(clusterJSONFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, cluster)
	require.NoError(t, err)

	cluster.RuntimeID = runtimeID
	cluster.RuntimeInput.Name = fmt.Sprintf("runtimeName%d", clusterVersion)
	cluster.Metadata.GlobalAccountID = fmt.Sprintf("globalAccountId%d", clusterVersion)
	cluster.KymaConfig.Profile = fmt.Sprintf("kymaProfile%d", clusterVersion)
	cluster.KymaConfig.Version = fmt.Sprintf("kymaVersion%d", clusterVersion)
	cluster.Kubeconfig = "fake kubeconfig"

	return cluster
}
