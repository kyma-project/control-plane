package reconciler

import (
	"strconv"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/pkg/errors"
)

/*
fakeClient is simulating API and db transactions in Reconciler Inventory

- registeredCluster is representation of 'inventory_clusters' table
  each unique clusterVersion should be a separate record
- registeredCluster.clusterConfigs is representation of `inventory_cluster_configs` table
  and it is a map[configVersion]Cluster; it stores different clusterConfigs for the same cluster
- registeredCluster.clusterStates is a map[configVersion]State; it simulates returning the status of given cluster in given configVersion
- registeredCluster.statusChanges is representation of 'inventory_cluster_config_statuses' table
  and it is a slice of *StatusChange; it contains all status changes for the cluster

calling RegisterCluster or UpdateCluster method on already existing cluster results in adding a new ClusterConfig

*/
type fakeClient struct {
	inventoryClusters map[string]*registeredCluster
}

type registeredCluster struct {
	clusterConfigs map[int64]Cluster
	clusterStates  map[int64]*State
	statusChanges  []*StatusChange
}

func NewFakeClient() *fakeClient {
	return &fakeClient{inventoryClusters: map[string]*registeredCluster{}}
}

// POST /v1/clusters
func (c *fakeClient) RegisterCluster(cluster Cluster) (*State, error) {
	return c.createOrUpdate(cluster)
}

// PUT /v1/clusters
func (c *fakeClient) UpdateCluster(cluster Cluster) (*State, error) {
	return c.createOrUpdate(cluster)
}

// DELETE /v1/clusters/{clusterName}
func (c *fakeClient) DeleteCluster(clusterName string) error {
	delete(c.inventoryClusters, clusterName)
	return nil
}

// GET /v1/clusters/{clusterName}/configs/{configVersion}/status
func (c *fakeClient) GetCluster(clusterName, configVersion string) (*State, error) {
	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &State{}, errors.New("not found")
	}
	v, err := strconv.ParseInt(configVersion, 10, 64)
	if err != nil {
		return &State{}, errors.New("invalid configVersion")
	}
	state, exists := existingCluster.clusterStates[v]
	if !exists {
		return &State{}, errors.New("not found")
	}
	return state, nil
}

// GET v1/clusters/{clusterName}/status
func (c *fakeClient) GetLatestCluster(clusterName string) (*State, error) {
	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &State{}, nil
	}
	latestConfigVersion := int64(len(existingCluster.clusterStates))

	return existingCluster.clusterStates[latestConfigVersion], nil
}

// GET v1/clusters/{clusterName}/statusChanges/{offset}
// offset is parsed to time.Duration
func (c *fakeClient) GetStatusChange(clusterName, offset string) ([]*StatusChange, error) {
	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return []*StatusChange{}, nil
	}
	return existingCluster.statusChanges, nil
}

func (c *fakeClient) createOrUpdate(cluster Cluster) (*State, error) {
	_, exists := c.inventoryClusters[cluster.Cluster]

	// initial creation call - cluster does not exist in db
	if !exists {
		c.inventoryClusters[cluster.Cluster] = &registeredCluster{
			clusterConfigs: map[int64]Cluster{
				1: cluster,
			},
			clusterStates: map[int64]*State{
				1: {
					Cluster:              cluster.Cluster,
					ClusterVersion:       1,
					ConfigurationVersion: 1,
					Status:               "reconcile_pending",
				},
			},
			statusChanges: append(c.inventoryClusters[cluster.Cluster].statusChanges, &StatusChange{
				Status:   ptr.String("reconcile_pending"),
				Duration: "10s",
			}),
		}

		return c.inventoryClusters[cluster.Cluster].clusterStates[1], nil
	}
	// cluster exists in db - add new configuration version
	//TODO: implement comparision mechanism for configs (new config should not be added if nothing changes in request) - needed for upgrade testing
    //TODO: implement clusterVersion bumping (happens when Kyma version is updated?)
	latestConfigVersion := int64(len(c.inventoryClusters[cluster.Cluster].clusterStates))
	c.inventoryClusters[cluster.Cluster].clusterStates[latestConfigVersion] = &State{
		Cluster:              cluster.Cluster,
		ClusterVersion:       1,
		ConfigurationVersion: latestConfigVersion + 1,
		Status:               "reconcile_pending",
	}
	c.inventoryClusters[cluster.Cluster].statusChanges = append(c.inventoryClusters[cluster.Cluster].statusChanges, &StatusChange{
		Status:   ptr.String("reconcile_pending"),
		Duration: "10s",
	})

	return c.inventoryClusters[cluster.Cluster].clusterStates[latestConfigVersion], nil
}

func (c *fakeClient) ChangeClusterState(clusterName string, clusterVersion int64, desiredState string) {
	c.inventoryClusters[clusterName].clusterStates[clusterVersion].Status = desiredState
	c.inventoryClusters[clusterName].statusChanges = append(c.inventoryClusters[clusterName].statusChanges, &StatusChange{
		Status:   ptr.String(desiredState),
		Duration: "10s",
	})
}
