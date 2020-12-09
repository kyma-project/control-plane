package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/tracing/opencensus"
	"go.opencensus.io/trace"
	"k8s.io/client-go/util/workqueue"

	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/kyma-project/control-plane/components/metris/internal/storage"
	"github.com/kyma-project/control-plane/components/metris/internal/tracing"
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
	// enable azure go-autorest tracing
	if tracing.IsEnabled() {
		if err := opencensus.Enable(); err != nil {
			config.Logger.With("error", err).Error("could not enable azure tracing")
		}
	}

	// retry after baseDelay*2^<num-failures>
	ratelimiter := workqueue.NewItemExponentialFailureRateLimiter(config.PollInterval, config.PollMaxInterval)

	return &Azure{
		config:           config,
		instanceStorage:  storage.NewMemoryStorage("clusters"),
		vmCapsStorage:    storage.NewMemoryStorage("vm_capabilities"),
		queue:            workqueue.NewNamedRateLimitingQueue(ratelimiter, "clients"),
		ClientAuthConfig: &DefaultAuthConfig{},
	}
}

// Run starts azure metrics gathering for all clusters returned by gardener.
func (a *Azure) Run(ctx context.Context) {
	a.config.Logger.Info("provider started")

	// remove throttling request (429) from the status codes for which the client will retry
	// this will help with rate limit issues
	autorest.StatusCodesForRetry = []int{
		http.StatusRequestTimeout, // 408
		// http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}

	go a.clusterHandler(ctx)

	var wg sync.WaitGroup

	wg.Add(a.config.Workers)

	for i := 0; i < a.config.Workers; i++ {
		go func(i int) {
			defer wg.Done()

			for {
				// lock till an item is available from the queue.
				clusterid, quit := a.queue.Get()
				workerlogger := a.config.Logger.With("worker", i).With("technicalid", clusterid)

				if quit {
					workerlogger.Debug("worker stopped")
					return
				}

				obj, ok := a.instanceStorage.Get(clusterid.(string))
				if !ok {
					workerlogger.Warn("cluster not found in storage, must have been instanceDeleted")
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
				rateLimited := a.processInstance(workerlogger, instance, ctx, getMetricsFromAzure)

				a.queue.Done(clusterid)
				if !rateLimited {
					a.queue.Forget(clusterid)
				}
				if a.queue.ShuttingDown() {
					workerlogger.Debugf("queue is shutting down, can't requeue cluster, processing cluster %s one last time", clusterid)
				} else {
					workerlogger.Debugf("enqueueing '%s'", clusterid)
					a.queue.AddRateLimited(clusterid)
				}
			}
		}(i)
	}

	wg.Wait()
	a.config.Logger.Info("provider stopped")
}

type metricsGetter func(context.Context, log.Logger, *Instance, *vmCapabilities, time.Duration, time.Duration) (*EventData, error)

func (a *Azure) processInstance(workerlogger log.Logger, instance *Instance, ctx context.Context, getMetrics metricsGetter) bool {
	var span *trace.Span
	if tracing.IsEnabled() {
		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/processInstance")
		defer span.End()

		workerlogger = workerlogger.With("traceID", span.SpanContext().TraceID).With("spanID", span.SpanContext().SpanID)
	}

	vmcaps := make(vmCapabilities)

	if obj, exists := a.vmCapsStorage.Get(instance.cluster.Region); exists {
		if caps, ok := obj.(*vmCapabilities); ok {
			vmcaps = *caps
		}
	} else {
		workerlogger.Warnf("vm capabilities for region %s not found, some metrics won't be available", instance.cluster.Region)
	}

	var (
		eventData       *EventData
		err             error
		rateLimited     bool
		instanceDeleted bool
	)

	eventData, err = getMetrics(ctx, workerlogger, instance, &vmcaps, a.config.PollInterval, a.config.PollingDuration)

	if err != nil {
		eventData, rateLimited, instanceDeleted = processError(ctx, workerlogger, instance, eventData, err, a.config.MaxRetries, a.instanceStorage)
	} else {
		span.Annotate(nil, "Reset retry attempts")
		instance.retryAttempts = 0
		a.instanceStorage.Put(instance.cluster.TechnicalID, instance)
	}

	if eventData != nil {
		if err := a.sendMetrics(ctx, workerlogger, instance, eventData); err != nil {
			workerlogger.With("error", err).Error("error parsing metric information, could not send eventData to EDP")
		}
		if !instanceDeleted {
			a.instanceStorage.Put(instance.cluster.TechnicalID, instance)
		}
	}
	return rateLimited
}

func processError(ctx context.Context, workerlogger log.Logger, instance *Instance, eventData *EventData, err error, maxRetries int, instanceStorage storage.Storage) (*EventData, bool, bool) {
	var span *trace.Span
	if tracing.IsEnabled() {
		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/processError")
		defer span.End()
	}
	eventData = instance.lastEvent
	workerlogger.With("error", err).Error("could not get metrics, using information from cache")
	if eventData == nil {
		workerlogger.With("error", err).Error("could not get metrics from cache, dropping events because no cached information")
	}

	rateLimited := false
	isDeleted := false
	if errdetail, ok := err.(autorest.DetailedError); ok {
		err = errdetail

		switch errdetail.StatusCode {
		// Check if the error is a resource group not found, then it would mean
		// that the cluster may have been instanceDeleted, and gardener did not trigger
		// the delete eventData or metris did not yet remove it from its cache.
		// Start retry attempt, then remove from storage if it reach max attempt.
		case http.StatusNotFound:
			span.Annotate(nil, "Status not found")
			if strings.Contains(errdetail.Original.Error(), ResponseErrCodeResourceGroupNotFound) {
				span.Annotate(nil, "ResourceGroup not found")
				span.Annotate(nil, "rate limiting")
				rateLimited = true
				instance.retryAttempts++
				if instance.retryAttempts < maxRetries {
					instanceStorage.Put(instance.cluster.TechnicalID, instance)
					workerlogger.Warnf("can't find resource group in azure, attempts: %d/%d", instance.retryAttempts, maxRetries)
					workerlogger.With("error", err).Warnf("resource group not found, retrying later")
				} else {
					span.Annotatef(nil, "Deleting cluster %v", instance.cluster.TechnicalID)
					instanceStorage.Delete(instance.cluster.TechnicalID)
					workerlogger.Warnf("removing cluster after %d attempts", maxRetries)
					isDeleted = true
				}
			} else {
				workerlogger.With("error", err).Warn("check error")
			}

		case http.StatusTooManyRequests:
			span.Annotate(nil, "Status too many requests")
			span.Annotate(nil, "rate limiting")
			workerlogger.With("error", err).Warn("received \"StatusTooManyRequests\", throttling")
			rateLimited = true

		default:
			span.Annotate(nil, "other error")
			workerlogger.With("error", err).Warn("check error")
		}

	}
	return eventData, rateLimited, isDeleted
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

			// if cluster was flag as instanceDeleted, remove it from storage and exit.
			if cluster.Deleted {
				logger.Info("removing cluster from storage")

				a.instanceStorage.Delete(cluster.TechnicalID)

				continue
			}

			instance := &Instance{cluster: cluster}

			// recover instance from storage.
			if obj, exists := a.instanceStorage.Get(cluster.TechnicalID); exists {
				if i, ok := obj.(*Instance); ok {
					instance.lastEvent = i.lastEvent
					instance.eventHubResourceGroupName = i.eventHubResourceGroupName
				}
			}

			// creating Azure REST API base client
			if client, err := newClient(cluster, logger, a.ClientAuthConfig); err != nil {
				logger.With("error", err).Error("error while creating client configuration, cluster will be ignored")
				a.instanceStorage.Delete(cluster.TechnicalID)

				continue
			} else {
				instance.client = client
			}

			if instance.eventHubResourceGroupName == "" {
				// Resource Groups for Event Hubs are tag with the subaccountid, if none is found, it may be a trial account.
				filter := fmt.Sprintf("tagname eq '%s' and tagvalue eq '%s'", tagNameSubAccountID, cluster.SubAccountID)

				if rg, err := instance.client.GetResourceGroup(parentctx, "", filter, logger); err != nil {
					if !cluster.Trial {
						logger.Warnf("could not find a resource group for eventData hub, cluster may not be ready, retrying in %s: %s", a.config.PollInterval, err)
						time.AfterFunc(a.config.PollInterval, func() { a.config.ClusterChannel <- cluster })

						continue
					}
				} else {
					instance.eventHubResourceGroupName = *rg.Name
				}
			}

			a.instanceStorage.Put(cluster.TechnicalID, instance)

			// initialize vm capabilities cache for the cluster region if not already.
			if _, exists := a.vmCapsStorage.Get(cluster.Region); !exists {
				logger.Debugf("initializing vm capabilities cache for region %s", instance.cluster.Region)
				filter := fmt.Sprintf("location eq '%s'", cluster.Region)

				var vmcaps = make(vmCapabilities) // [vmtype][capname]capvalue

				if skuList, err := instance.client.GetVMResourceSkus(parentctx, filter, logger); err != nil {
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

			a.queue.Add(cluster.TechnicalID)
		case <-parentctx.Done():
			a.config.Logger.Debug("stopping cluster handler")
			a.queue.ShutDown()

			return
		}
	}
}

// getMetrics - collect results from different Azure API and create edp events.
func getMetricsFromAzure(parentctx context.Context, workerlogger log.Logger, instance *Instance, vmcaps *vmCapabilities, pollInterval time.Duration, pollingDuration time.Duration) (*EventData, error) {
	if tracing.IsEnabled() {
		var span *trace.Span

		parentctx, span = trace.StartSpan(parentctx, "metris/provider/azure/getMetrics")
		defer span.End()

		workerlogger = workerlogger.With("traceID", span.SpanContext().TraceID).With("spanID", span.SpanContext().SpanID)
	}

	workerlogger.Debug("getting metrics")

	// Using a timeout context to prevent azure api to hang for too long,
	// sometimes client get stuck waiting even with a max poll duration is set.
	// If it reach the time limit, last successful eventData data will be returned.
	ctx, cancel := context.WithTimeout(parentctx, pollingDuration)
	defer cancel()

	computeData, err := instance.getComputeMetrics(ctx, workerlogger, vmcaps)
	if err != nil {
		return nil, err
	}

	networkData, err := instance.getNetworkMetrics(ctx, workerlogger)
	if err != nil {
		return nil, err
	}

	eventData := &EventData{
		ResourceGroups: []string{instance.cluster.TechnicalID},
		Compute:        computeData,
		Networking:     networkData,
		// init an empty eventhub data, because they are optional (trial account)
		EventHub: &EventHub{
			NumberNamespaces:     0,
			IncomingRequestsPT1M: 0,
			MaxIncomingBytesPT1M: 0,
			MaxOutgoingBytesPT1M: 0,
			IncomingRequestsPT5M: 0,
			MaxIncomingBytesPT5M: 0,
			MaxOutgoingBytesPT5M: 0,
		},
	}

	if len(instance.eventHubResourceGroupName) > 0 {
		eventhubData, err := instance.getEventHubMetrics(ctx, pollInterval, workerlogger)
		if err != nil {
			return nil, err
		}

		eventData.ResourceGroups = append(eventData.ResourceGroups, instance.eventHubResourceGroupName)
		eventData.EventHub = eventhubData
	}

	return eventData, nil
}

// sendMetrics - send events to EDP.
func (a *Azure) sendMetrics(ctx context.Context, workerlogger log.Logger, instance *Instance, eventData *EventData) error {
	if tracing.IsEnabled() {
		var span *trace.Span
		ctx, span = trace.StartSpan(ctx, "metris/provider/azure/sendMetrics")
		defer span.End()
	}
	eventDataRaw, err := json.Marshal(&eventData)
	if err != nil {
		return err
	}

	// save a copy of the eventData data in case of error next time
	instance.lastEvent = eventData

	eventDataJSON := json.RawMessage(eventDataRaw)

	eventBuffer := edp.Event{
		Datatenant: instance.cluster.SubAccountID,
		Data:       &eventDataJSON,
	}

	workerlogger.Debug("sending eventData to EDP")

	a.config.EventsChannel <- &eventBuffer

	return nil
}
