package reconciler

import (
	"sync"
	"time"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
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
	clusterConfigs map[int64]reconcilerApi.Cluster
	clusterStates  map[int64]*reconcilerApi.HTTPClusterResponse
	statusChanges  []*reconcilerApi.StatusChange
}

func NewFakeClient() *FakeClient {
	return &FakeClient{inventoryClusters: map[string]*registeredCluster{}, deleted: map[string]struct{}{}}
}

// POST /v1/clusters
func (c *FakeClient) ApplyClusterConfig(cluster reconcilerApi.Cluster) (*reconcilerApi.HTTPClusterResponse, error) {
	return c.addToInventory(cluster)
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
func (c *FakeClient) GetCluster(clusterName string, configVersion int64) (*reconcilerApi.HTTPClusterResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &reconcilerApi.HTTPClusterResponse{}, errors.New("not found")
	}
	state, exists := existingCluster.clusterStates[configVersion]
	if !exists {
		return &reconcilerApi.HTTPClusterResponse{}, errors.New("not found")
	}
	return state, nil
}

// GET v1/clusters/{clusterName}/status
func (c *FakeClient) GetLatestCluster(clusterName string) (*reconcilerApi.HTTPClusterResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return &reconcilerApi.HTTPClusterResponse{}, nil
	}
	latestConfigVersion := int64(len(existingCluster.clusterStates))

	return existingCluster.clusterStates[latestConfigVersion], nil
}

// GET v1/clusters/{clusterName}/statusChanges/{offset}
// offset is parsed to time.Duration
func (c *FakeClient) GetStatusChange(clusterName, offset string) ([]*reconcilerApi.StatusChange, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingCluster, exists := c.inventoryClusters[clusterName]
	if !exists {
		return []*reconcilerApi.StatusChange{}, nil
	}
	return existingCluster.statusChanges, nil
}

func (c *FakeClient) addToInventory(cluster reconcilerApi.Cluster) (*reconcilerApi.HTTPClusterResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.inventoryClusters[cluster.RuntimeID]

	// initial creation call - cluster does not exist in db
	if !exists {
		c.inventoryClusters[cluster.RuntimeID] = &registeredCluster{
			clusterConfigs: map[int64]reconcilerApi.Cluster{
				1: cluster,
			},
			clusterStates: map[int64]*reconcilerApi.HTTPClusterResponse{
				1: {
					Cluster:              cluster.RuntimeID,
					ClusterVersion:       1,
					ConfigurationVersion: 1,
					Status:               "reconcile_pending",
				},
			},
			statusChanges: []*reconcilerApi.StatusChange{{
				Status:   reconcilerApi.StatusReconcilePending,
				Duration: int64(10 * time.Second),
			}},
		}

		return c.inventoryClusters[cluster.RuntimeID].clusterStates[1], nil
	}
	// cluster exists in db - add new configuration version
	latestConfigVersion := int64(len(c.inventoryClusters[cluster.RuntimeID].clusterStates)) + 1
	c.inventoryClusters[cluster.RuntimeID].clusterStates[latestConfigVersion] = &reconcilerApi.HTTPClusterResponse{
		Cluster:              cluster.RuntimeID,
		ClusterVersion:       1,
		ConfigurationVersion: latestConfigVersion,
		Status:               "reconcile_pending",
	}
	c.inventoryClusters[cluster.RuntimeID].statusChanges = append(c.inventoryClusters[cluster.RuntimeID].statusChanges, &reconcilerApi.StatusChange{
		Status:   reconcilerApi.StatusReconcilePending,
		Duration: int64(10 * time.Second),
	})
	c.inventoryClusters[cluster.RuntimeID].clusterConfigs[latestConfigVersion] = cluster

	return c.inventoryClusters[cluster.RuntimeID].clusterStates[latestConfigVersion], nil
}

func (c *FakeClient) ChangeClusterState(clusterName string, clusterVersion int64, desiredState reconcilerApi.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inventoryClusters[clusterName].clusterStates[clusterVersion].Status = desiredState
	c.inventoryClusters[clusterName].statusChanges = append(c.inventoryClusters[clusterName].statusChanges, &reconcilerApi.StatusChange{
		Status:   desiredState,
		Duration: int64(10 * time.Second),
	})
}

func (c *FakeClient) ChangeClusterStateForAllReconciliations(desiredState reconcilerApi.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cluster := range c.inventoryClusters {
		for _, clusterState := range cluster.clusterStates {
			clusterState.Status = desiredState
		}
	}
}

func (c *FakeClient) LastClusterConfig(runtimeID string) (*reconcilerApi.Cluster, error) {
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

func getLastClusterConfig(cluster *registeredCluster) (*reconcilerApi.Cluster, error) {
	clusterConfig, found := cluster.clusterConfigs[int64(len(cluster.clusterConfigs))]
	if !found {
		return nil, errors.New("cluster config not found in cluster configs inventory")
	}
	return &clusterConfig, nil
}
