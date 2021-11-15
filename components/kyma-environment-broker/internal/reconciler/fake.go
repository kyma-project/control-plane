package reconciler

import (
	"sync"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/pkg/errors"
)

/*
FakeClient is simulating API and db transactions in Reconciler Inventory

- registeredCluster is representation of 'inventory_clusters' table
  each unique clusterVersion should be a separate record
- registeredCluster.clusterConfigs is representation of `inventory_cluster_configs` table
  and it is a map[configVersion]Cluster; it stores different clusterConfigs for the same cluster
- registeredCluster.clusterStates is a map[configVersion]State; it simulates returning the status of given cluster in given configVersion
- registeredCluster.statusChanges is representation of 'inventory_cluster_config_statuses' table
  and it is a slice of *StatusChange; it contains all status changes for the cluster

calling ApplyClusterConfig method on already existing cluster results in adding a new ClusterConfig

*/
type FakeClient struct {
	mu                sync.Mutex
	inventoryClusters map[string]*registeredCluster
	deleted           map[string]struct{}
}

type registeredCluster struct {
	clusterConfigs map[int64]Cluster
	clusterStates  map[int64]*State
	statusChanges  []*StatusChange
}

func NewFakeClient() *FakeClient {
	return &FakeClient{inventoryClusters: map[string]*registeredCluster{}, deleted: map[string]struct{}{}}
}

// POST /v1/clusters
func (c *FakeClient) ApplyClusterConfig(cluster Cluster) (*State, error) {
	return c.createOrUpdate(cluster)
}

// DELETE /v1/clusters/{clusterName}
func (c *FakeClient) DeleteCluster(clusterName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.inventoryClusters[clusterName]
	if !exists {
		return nil
	}
	c.deleted[clusterName] = struct{}{}
	return nil
}

// GET /v1/clusters/{clusterName}/configs/{configVersion}/status
func (c *FakeClient) GetCluster(clusterName string, configVersion int64) (*State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &State{}, errors.New("not found")
	}
	state, exists := existingCluster.clusterStates[configVersion]
	if !exists {
		return &State{}, errors.New("not found")
	}
	return state, nil
}

// GET v1/clusters/{clusterName}/status
func (c *FakeClient) GetLatestCluster(clusterName string) (*State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &State{}, nil
	}
	latestConfigVersion := int64(len(existingCluster.clusterStates))

	return existingCluster.clusterStates[latestConfigVersion], nil
}

// GET v1/clusters/{clusterName}/statusChanges/{offset}
// offset is parsed to time.Duration
func (c *FakeClient) GetStatusChange(clusterName, offset string) ([]*StatusChange, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return []*StatusChange{}, nil
	}
	return existingCluster.statusChanges, nil
}

func (c *FakeClient) createOrUpdate(cluster Cluster) (*State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
			statusChanges: []*StatusChange{{
				Status:   ptr.String("reconcile_pending"),
				Duration: "10s",
			}},
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

func (c *FakeClient) ChangeClusterState(clusterName string, clusterVersion int64, desiredState string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inventoryClusters[clusterName].clusterStates[clusterVersion].Status = desiredState
	c.inventoryClusters[clusterName].statusChanges = append(c.inventoryClusters[clusterName].statusChanges, &StatusChange{
		Status:   ptr.String(desiredState),
		Duration: "10s",
	})
}

func (c *FakeClient) LastClusterConfig(runtimeID string) (*Cluster, error) {
	cluster, found := c.inventoryClusters[runtimeID]
	if !found {
		return nil, errors.New("cluster not found in clusters inventory")
	}
	return getLastClusterConfig(cluster)
}

func (c *FakeClient) IsBeingDeleted(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.deleted[id]
	if exists {
		return true
	}

	return false
}

func (c *FakeClient) ClusterExists(id string) bool {
	_, found := c.inventoryClusters[id]
	return found
}

func getLastClusterConfig(cluster *registeredCluster) (*Cluster, error) {
	clusterConfig, found := cluster.clusterConfigs[int64(1)]
	if !found {
		return nil, errors.New("cluster config not found in cluster configs inventory")
	}
	return &clusterConfig, nil
}
