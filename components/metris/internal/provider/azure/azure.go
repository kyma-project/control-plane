package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/kyma-project/control-plane/components/metris/internal/storage"
	"k8s.io/client-go/util/workqueue"
)

var (
	// register the azure provider
	_ = func() struct{} {
		err := provider.RegisterProvider("az", NewAzureProvider)
		if err != nil {
			panic(err)
		}
		return struct{}{}
	}()
)

// NewAzureProvider returns a new Azure provider.
func NewAzureProvider(config *provider.Config) provider.Provider {
	return &Azure{
		config:           config,
		instanceStorage:  storage.NewMemoryStorage(),
		vmCapsStorage:    storage.NewMemoryStorage(),
		queue:            workqueue.NewNamedDelayingQueue("clients"),
		ClientAuthConfig: &DefaultAuthConfig{},
	}
}

// Run starts azure metrics gathering for all clusters returned by gardener.
func (a *Azure) Run(ctx context.Context) {
	a.config.Logger.Info("provider started")

	go a.clusterHandler(ctx)

	var wg sync.WaitGroup

	wg.Add(a.config.Workers)

	for i := 0; i < a.config.Workers; i++ {
		go func(i int) {
			defer wg.Done()

			for {
				// lock till a item is available from the queue.
				clusterid, quit := a.queue.Get()
				workerlogger := a.config.Logger.With("worker", i).With("technicalid", clusterid)

				if quit {
					workerlogger.Debug("worker stopped")
					return
				}

				obj, ok := a.instanceStorage.Get(clusterid.(string))
				if !ok {
					workerlogger.Warn("cluster not found in storage, must have been deleted")
					a.queue.Done(clusterid)

					continue
				}

				instance, ok := obj.(*Instance)
				if !ok {
					workerlogger.Error("cluster object is corrupted, removing it from storage")
					a.instanceStorage.Delete(clusterid.(string))
					a.queue.Done(clusterid)

					continue
				}

				workerlogger = workerlogger.With("account", instance.cluster.AccountID).With("subaccount", instance.cluster.SubAccountID)

				vmcaps := make(vmCapabilities)

				if obj, exists := a.vmCapsStorage.Get(instance.cluster.Region); exists {
					if caps, ok := obj.(*vmCapabilities); ok {
						vmcaps = *caps
					}
				} else {
					workerlogger.Warnf("vm capabilities for region %s not found, some metrics won't be available", instance.cluster.Region)
				}

				a.gatherMetrics(ctx, workerlogger, instance, &vmcaps)

				a.queue.Done(clusterid)

				// requeue item after X duration if client still in storage
				if !a.queue.ShuttingDown() {
					if _, exists := a.instanceStorage.Get(clusterid.(string)); exists {
						workerlogger.Debugf("requeuing cluster in %s", a.config.PollInterval)
						a.queue.AddAfter(clusterid, a.config.PollInterval)
					} else {
						workerlogger.Warn("can't requeue cluster, must have been deleted")
					}
				} else {
					workerlogger.Debug("queue is shutting down, can't requeue cluster")
				}
			}
		}(i)
	}

	wg.Wait()
	a.config.Logger.Info("provider stopped")
}

// clusterHandler listen on the cluster channel then update the storage and the queue.
func (a *Azure) clusterHandler(parentctx context.Context) {
	a.config.Logger.Debug("starting cluster handler")

	for {
		select {
		case cluster := <-a.config.ClusterChannel:
			logger := a.config.Logger.
				With("technicalid", cluster.TechnicalID).
				With("accountid", cluster.AccountID).
				With("subaccountid", cluster.SubAccountID)

			logger.Debug("received cluster from gardener controller")

			// if cluster was flag as deleted, remove it from storage and exit.
			if cluster.Deleted {
				logger.Info("removing cluster")

				a.instanceStorage.Delete(cluster.TechnicalID)

				continue
			}

			// creating Azure REST API base client
			client, err := newClient(cluster, logger, a.config.ClientTraceLevel, a.ClientAuthConfig)
			if err != nil {
				logger.With("error", err).Error("error while creating client configuration")
				continue
			}

			instance := &Instance{
				cluster:   cluster,
				client:    client,
				lastEvent: &EventData{},
			}

			// getting cluster and event hub resource group names.
			rg, err := client.GetResourceGroup(parentctx, cluster.TechnicalID, "", logger)
			if err != nil {
				logger.Errorf("could not find cluster resource group, cluster may not be ready, retrying in %s: %s", a.config.PollInterval, err)
				time.AfterFunc(a.config.PollInterval, func() { a.config.ClusterChannel <- cluster })

				continue
			} else {
				instance.clusterResourceGroupName = *rg.Name
			}

			// Resource Groups for Event Hubs are tag with the subaccountid, if none is found, it may be a trial account.
			filter := fmt.Sprintf("tagname eq '%s' and tagvalue eq '%s'", tagNameSubAccountID, cluster.SubAccountID)

			rg, err = client.GetResourceGroup(parentctx, "", filter, logger)
			if err != nil {
				if cluster.Trial {
					logger.Warn("trial cluster, could not find event hubs resource groups, metrics will not be reported for the event hubs")
				} else {
					logger.Errorf("could not find event hub resource groups, cluster may not be ready, retrying in %s: %s", a.config.PollInterval, err)
					time.AfterFunc(a.config.PollInterval, func() { a.config.ClusterChannel <- cluster })

					continue
				}
			} else {
				instance.eventHubResourceGroupName = *rg.Name
			}

			// recover the last event value if cluster already exists in storage.
			obj, exists := a.instanceStorage.Get(cluster.TechnicalID)
			if exists {
				if i, ok := obj.(*Instance); ok {
					instance.lastEvent = i.lastEvent
				}
			}

			// initialize vm capabilities cache for the cluster region if not already.
			_, exists = a.vmCapsStorage.Get(cluster.Region)
			if !exists {
				logger.Debugf("initializing vm capabilities cache for region %s", instance.cluster.Region)
				filter := fmt.Sprintf("location eq '%s'", cluster.Region)

				var vmcaps = make(vmCapabilities) // [vmtype][capname]capvalue

				skuList, err := instance.client.GetVMResourceSkus(parentctx, filter)
				if err != nil {
					logger.Errorf("error while getting vm capabilities for region %s: %s", cluster.Region, err)
				} else {
					for _, item := range skuList {
						vmcaps[*item.Name] = make(map[string]string)
						for _, v := range *item.Capabilities {
							vmcaps[*item.Name][*v.Name] = *v.Value
						}
					}
				}

				if len(vmcaps) > 0 {
					a.vmCapsStorage.Put(instance.cluster.Region, &vmcaps)
				}
			}

			a.instanceStorage.Put(cluster.TechnicalID, instance)

			a.queue.Add(cluster.TechnicalID)
		case <-parentctx.Done():
			a.config.Logger.Debug("stopping cluster handler")
			a.queue.ShutDown()

			return
		}
	}
}

// gatherMetrics - collect results from different Azure API and create edp events.
func (a *Azure) gatherMetrics(parentctx context.Context, workerlogger log.Logger, instance *Instance, vmcaps *vmCapabilities) {
	var (
		cluster    = instance.cluster
		datatenant = cluster.SubAccountID
	)

	resourceGroupName := instance.clusterResourceGroupName
	eventHubResourceGroupName := instance.eventHubResourceGroupName

	eventData := &EventData{ResourceGroups: []string{resourceGroupName}}
	if len(eventHubResourceGroupName) > 0 {
		eventData.ResourceGroups = append(eventData.ResourceGroups, eventHubResourceGroupName)
	}

	workerlogger.Debug("getting metrics")

	// Using a timeout context to prevent azure api to hang for too long,
	// sometimes client get stuck waiting even with a max poll duration of 1 min.
	// If it reach the time limit, last successful event data will be returned.
	ctx, cancel := context.WithTimeout(parentctx, maxPollingDuration)
	defer cancel()

	eventData.Compute = instance.getComputeMetrics(ctx, resourceGroupName, workerlogger, vmcaps)
	eventData.Networking = instance.getNetworkMetrics(ctx, resourceGroupName, workerlogger)
	eventData.EventHub = instance.getEventHubMetrics(ctx, a.config.PollInterval, eventHubResourceGroupName, workerlogger)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		workerlogger.Warn("Azure REST API call timedout, sending last successful event data")

		if instance.lastEvent == nil {
			return
		}

		eventData.Compute = instance.lastEvent.Compute
		eventData.Networking = instance.lastEvent.Networking
		eventData.EventHub = instance.lastEvent.EventHub
	}

	eventDataRaw, err := json.Marshal(&eventData)
	if err != nil {
		workerlogger.Errorf("error parsing azure events to json, could not send event to EDP: %s", err)
		return
	}

	// save a copy of the event data in case of error next time
	instance.lastEvent = eventData

	eventDataJSON := json.RawMessage(eventDataRaw)

	eventBuffer := edp.Event{
		Datatenant: datatenant,
		Data:       &eventDataJSON,
	}

	a.config.EventsChannel <- &eventBuffer
}
