package process

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"
	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"
	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"
)

type Process struct {
	KEBClient         *keb.Client
	EDPClient         *edp.Client
	Queue             workqueue.DelayingInterface
	SecretCacheClient v1.CoreV1Interface
	Cache             *cache.Cache
	Providers         *Providers
	ScrapeInterval    time.Duration
	WorkersPoolSize   int
	NodeConfig        skrnode.ConfigInf
	PVCConfig         skrpvc.ConfigInf
	SvcConfig         skrsvc.ConfigInf
	Logger            *zap.SugaredLogger
}

var (
	errorSubAccountIDNotTrackable = errors.New("subAccountID is not trackable")
	ErrLoadingFailed              = errors.New("could not load resource")
)

const (
	trackableTrue  = true
	trackableFalse = false
)

func (p *Process) generateRecordWithNewMetrics(identifier int, subAccountID string) (kmccache.Record, error) {
	ctx := context.Background()
	var ok bool

	obj, isFound := p.Cache.Get(subAccountID)
	if !isFound {
		err := errorSubAccountIDNotTrackable
		return kmccache.Record{}, err
	}

	var record kmccache.Record
	if record, ok = obj.(kmccache.Record); !ok {
		err := fmt.Errorf("bad item from cache, could not cast to a record obj")
		return kmccache.Record{}, err
	}
	p.namedLogger().With(log.KeyWorkerID, identifier).Debugf("record found from cache: %+v", record)

	runtimeID := record.RuntimeID

	kubeconfig, err := kmccache.GetKubeConfigFromCache(p.Logger, p.SecretCacheClient, runtimeID)
	if err != nil {
		return record, fmt.Errorf("loading Kubeconfig for %s failed: %w", ErrLoadingFailed, err)
	}
	record.KubeConfig = kubeconfig

	// Get nodes dynamic client
	nodesClient, err := p.NodeConfig.NewClient(record)
	if err != nil {
		return record, err
	}

	// Get nodes
	var nodes *corev1.NodeList
	nodes, err = nodesClient.List(ctx)
	if err != nil {
		return record, err
	}

	if len(nodes.Items) == 0 {
		err = fmt.Errorf("no nodes to process")
		return record, err
	}

	// Get PVCs
	pvcClient, err := p.PVCConfig.NewClient(record)
	if err != nil {
		return record, err
	}
	var pvcList *corev1.PersistentVolumeClaimList
	pvcList, err = pvcClient.List(ctx)
	if err != nil {
		return record, err
	}

	// Get Svcs
	var svcList *corev1.ServiceList
	svcClient, err := p.SvcConfig.NewClient(record)
	if err != nil {
		return record, err
	}
	svcList, err = svcClient.List(ctx)
	if err != nil {
		return record, err
	}

	// Create input
	input := Input{
		provider: record.ProviderType,
		nodeList: nodes,
		pvcList:  pvcList,
		svcList:  svcList,
	}
	metric, err := input.Parse(p.Providers)
	if err != nil {
		return record, err
	}
	metric.RuntimeId = record.RuntimeID
	metric.SubAccountId = record.SubAccountID
	metric.ShootName = record.ShootName
	record.Metric = metric
	return record, nil
}

// getOldRecordIfMetricExists gets old record from cache if old metric exists.
func (p *Process) getOldRecordIfMetricExists(subAccountID string) (*kmccache.Record, error) {
	oldRecordObj, found := p.Cache.Get(subAccountID)
	if !found {
		notFoundErr := fmt.Errorf("subAccountID: %s not found", subAccountID)
		p.Logger.Error(notFoundErr)
		return nil, notFoundErr
	}

	if oldRecord, ok := oldRecordObj.(kmccache.Record); ok {
		if oldRecord.Metric != nil {
			return &oldRecord, nil
		}
	}
	notFoundErr := fmt.Errorf("old metrics for subAccountID: %s not found", subAccountID)
	p.Logger.With(log.KeySubAccountID, subAccountID).Error("old metrics for subAccount not found")
	return nil, notFoundErr
}

// pollKEBForRuntimes polls KEB for runtimes information.
func (p *Process) pollKEBForRuntimes() {
	kebReq, err := p.KEBClient.NewRequest()
	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			Fatal("create a new request for KEB")
	}
	for {
		runtimesPage, err := p.KEBClient.GetAllRuntimes(kebReq)
		if err != nil {
			p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
				Error("get runtimes from KEB")
			time.Sleep(p.KEBClient.Config.PollWaitDuration)
			continue
		}

		p.namedLogger().Debugf("num of runtimes are: %d", runtimesPage.Count)
		p.populateCacheAndQueue(runtimesPage)
		p.namedLogger().Debugf("length of the cache after KEB is done populating: %d", p.Cache.ItemCount())
		p.namedLogger().Infof("waiting to poll KEB again after %v....", p.KEBClient.Config.PollWaitDuration)
		time.Sleep(p.KEBClient.Config.PollWaitDuration)
	}
}

// Start runs the complete process of collection and sending metrics.
func (p *Process) Start() {
	var wg sync.WaitGroup
	go func() {
		p.pollKEBForRuntimes()
	}()

	for i := 0; i < p.WorkersPoolSize; i++ {
		j := i
		go func() {
			defer wg.Done()
			p.execute(j)
			p.namedLogger().Debugf("########  Worker exits ########")
		}()
	}
	wg.Wait()
}

// Execute is executed by each worker to process an entry from the queue.
func (p *Process) execute(identifier int) {
	for {
		// Pick up a subAccountID to process from queue and mark as Done()
		subAccountIDObj, _ := p.Queue.Get()
		subAccountID := fmt.Sprintf("%v", subAccountIDObj)

		// TODO Implement cleanup holistically in #kyma-project/control-plane/issues/512
		// if isShuttingDown {
		//	//p.Cleanup()
		//	return
		//}

		p.processSubAccountID(subAccountID, identifier)
		p.Queue.Done(subAccountIDObj)
	}
}

func (p *Process) processSubAccountID(subAccountID string, identifier int) {
	var payload []byte
	if strings.TrimSpace(subAccountID) == "" {
		p.namedLogger().With(log.KeyWorkerID, identifier).Warn("cannot work with empty subAccountID")

		// Nothing to do further
		return
	}
	p.namedLogger().With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
		Debug("fetched subAccountID from queue")

	record, isOldMetricValid, err := p.getRecordWithOldOrNewMetric(identifier, subAccountID)
	if err != nil {
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).Error("no metric found/generated for subaccount")
		// SubAccountID is not trackable anymore as there is no runtime
		if errors.Is(err, errorSubAccountIDNotTrackable) {
			p.namedLoggerWithRuntime(record).With(log.KeyRequeue, log.ValueFalse).With(log.KeySubAccountID, subAccountID).
				With(log.KeyWorkerID, identifier).Info("subAccountID requeued")
			return
		}
		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRuntime(record).With(log.KeyRequeue, log.ValueTrue).With(log.KeySubAccountID, subAccountID).
			With(log.KeyWorkerID, identifier).Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		if oldRecord := p.getSubAccountFromCache(subAccountID); oldRecord != nil {
			recordSubAccountProcessed(false, *oldRecord)
		}

		// Nothing to do further
		return
	}

	// Convert metric to JSON
	payload, err = json.Marshal(*record.Metric)
	if err != nil {
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
			Error("json.Marshal metric for subAccountID")

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
			With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
			Debugf("requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		recordSubAccountProcessed(false, *record)

		// Nothing to do further
		return
	}

	// Send metrics to EDP
	// Note: EDP refers SubAccountID as tenant
	p.namedLoggerWithRuntime(record).With(log.KeySubAccountID, subAccountID).
		With(log.KeyWorkerID, identifier).Debugf("sending EventStreamToEDP: payload: %s", string(payload))
	err = p.sendEventStreamToEDP(subAccountID, payload)
	if err != nil {
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
			Errorf("send metric to EDP for event-stream: %s", string(payload))

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
			With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
			Debugf("requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		recordSubAccountProcessed(false, *record)

		// Nothing to do further hence continue
		return
	}
	p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueSuccess).With(log.KeySubAccountID, subAccountID).
		With(log.KeyWorkerID, identifier).Infof("sent event stream, shoot: %s", record.ShootName)

	// record metrics.
	recordSubAccountProcessed(true, *record)
	recordSubAccountProcessedTimeStamp(isOldMetricValid, *record)

	// update cache.
	if !isOldMetricValid {
		p.Cache.Set(record.SubAccountID, *record, cache.NoExpiration)
		p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueSuccess).With(log.KeySubAccountID, record.SubAccountID).
			With(log.KeyWorkerID, identifier).Debug("saved metric")
		resetOldMetricsPublishedGauge(*record)
	} else {
		// record metric.
		recordOldMetricsPublishedGauge(*record)
	}

	// Requeue the subAccountID anyway
	p.namedLoggerWithRuntime(record).With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
		With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
		Debugf("requeued subAccountID after %v", p.ScrapeInterval)
	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
}

// getRecordWithOldOrNewMetric generates new metric or fetches the old metric along with a bool flag which
// indicates whether it is an old metric or not(true, when it is old and false when it is new).
func (p *Process) getRecordWithOldOrNewMetric(identifier int, subAccountID string) (*kmccache.Record, bool, error) {
	record, err := p.generateRecordWithNewMetrics(identifier, subAccountID)
	if err != nil {
		if errors.Is(err, errorSubAccountIDNotTrackable) {
			p.namedLoggerWithRuntime(&record).With(log.KeySubAccountID, subAccountID).
				With(log.KeyWorkerID, identifier).Info("subAccountID is not trackable anymore, skipping the fetch of old metric")
			return nil, false, err
		}
		p.namedLoggerWithRuntime(&record).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).With(log.KeySubAccountID, subAccountID).
			Error("generate new metric for subAccount")
		// Get old data
		oldRecord, err := p.getOldRecordIfMetricExists(subAccountID)
		if err != nil {
			// Nothing to do
			return nil, false, errors.Wrapf(err, "failed to get getOldMetric for subaccountID: %s", subAccountID)
		}
		return oldRecord, true, nil
	}
	return &record, false, nil
}

func (p *Process) sendEventStreamToEDP(tenant string, payload []byte) error {
	edpRequest, err := p.EDPClient.NewRequest(tenant)
	if err != nil {
		return errors.Wrapf(err, "failed to create a new request for EDP")
	}

	resp, err := p.EDPClient.Send(edpRequest, payload)
	if err != nil {
		return errors.Wrapf(err, "failed to send event-stream to EDP")
	}

	if !isSuccess(resp.StatusCode) {
		return fmt.Errorf("failed to send event-stream to EDP as it returned HTTP: %d", resp.StatusCode)
	}
	return nil
}

func isSuccess(status int) bool {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return true
	}
	return false
}

// isTrackableState returns true if the runtime state is trackable, otherwise returns false.
func isTrackableState(state kebruntime.State) bool {
	switch state {
	case kebruntime.StateSucceeded, kebruntime.StateError, kebruntime.StateUpgrading, kebruntime.StateUpdating:
		return true
	}
	return false
}

// isProvisionedStatus returns true if the runtime is successfully provisioned, otherwise returns false.
func isProvisionedStatus(runtime kebruntime.RuntimeDTO) bool {
	if runtime.Status.Provisioning != nil &&
		runtime.Status.Provisioning.State == string(kebruntime.StateSucceeded) &&
		runtime.Status.Deprovisioning == nil {
		return true
	}
	return false
}

func isRuntimeTrackable(runtime kebruntime.RuntimeDTO) bool {
	return isTrackableState(runtime.Status.State) || isProvisionedStatus(runtime)
}

// getOrDefault returns the runtime state or a default value if runtimeStatus is nil
func getOrDefault(runtimeStatus *kebruntime.Operation, defaultValue string) string {
	if runtimeStatus != nil {
		return runtimeStatus.State
	}
	return defaultValue
}

// populateCacheAndQueue populates Cache and Queue with new runtimes and deletes the runtimes which should not be tracked.
func (p *Process) populateCacheAndQueue(runtimes *kebruntime.RuntimesPage) {
	// clear the gauge to fill it with the new data
	kebFetchedClusters.Reset()

	validSubAccounts := make(map[string]bool)
	for _, runtime := range runtimes.Data {
		if runtime.SubAccountID == "" {
			continue
		}
		validSubAccounts[runtime.SubAccountID] = true
		recordObj, isFoundInCache := p.Cache.Get(runtime.SubAccountID)

		// Get provisioning and deprovisioning states if available otherwise return empty string for logging.
		provisioning := getOrDefault(runtime.Status.Provisioning, "")
		deprovisioning := getOrDefault(runtime.Status.Deprovisioning, "")
		p.namedLogger().
			With(log.KeySubAccountID, runtime.SubAccountID).
			With(log.KeyRuntimeID, runtime.RuntimeID).
			With(log.KeyRuntimeState, runtime.Status.State).
			With(log.KeyProvisioningStatus, provisioning).
			With(log.KeyDeprovisioningStatus, deprovisioning).
			Info("Runtime state")

		if isRuntimeTrackable(runtime) {
			newRecord := kmccache.Record{
				SubAccountID:    runtime.SubAccountID,
				RuntimeID:       runtime.RuntimeID,
				InstanceID:      runtime.InstanceID,
				GlobalAccountID: runtime.GlobalAccountID,
				ShootName:       runtime.ShootName,
				ProviderType:    strings.ToLower(runtime.Provider),
				KubeConfig:      "",
				Metric:          nil,
			}

			// record kebFetchedClusters metric for trackable cluster
			recordKEBFetchedClusters(
				trackableTrue,
				runtime.ShootName,
				runtime.InstanceID,
				runtime.RuntimeID,
				runtime.SubAccountID,
				runtime.GlobalAccountID)

			// Cluster is trackable but does not exist in the cache
			if !isFoundInCache {
				err := p.Cache.Add(runtime.SubAccountID, newRecord, cache.NoExpiration)
				if err != nil {
					p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
						With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyRuntimeID, runtime.RuntimeID).
						Error("Failed to add subAccountID to cache hence skipping queueing it")
					continue
				}
				p.Queue.Add(runtime.SubAccountID)
				p.namedLogger().With(log.KeyResult, log.ValueSuccess).With(log.KeySubAccountID, runtime.SubAccountID).
					With(log.KeyRuntimeID, runtime.RuntimeID).Debug("Queued and added to cache")
				continue
			}

			// Cluster is trackable and exists in the cache
			if record, ok := recordObj.(kmccache.Record); ok {
				if record.ShootName != runtime.ShootName {
					// The shootname has changed hence the record in the cache is not valid anymore
					// No need to queue as the subAccountID already exists in queue
					p.Cache.Set(runtime.SubAccountID, newRecord, cache.NoExpiration)
					p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyRuntimeID, runtime.RuntimeID).
						Debug("Resetted the values in cache for subAccount")

					// delete metrics for old shoot name.
					if success := deleteMetrics(record); !success {
						p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyRuntimeID, runtime.RuntimeID).
							Info("prometheus metrics were not successfully removed for subAccount")
					}
				}
			}
			continue
		}

		// record kebFetchedClusters metric for not trackable clusters
		recordKEBFetchedClusters(
			trackableFalse,
			runtime.ShootName,
			runtime.InstanceID,
			runtime.RuntimeID,
			runtime.SubAccountID,
			runtime.GlobalAccountID)

		if isFoundInCache {
			// Cluster is not trackable but is found in cache should be deleted
			p.Cache.Delete(runtime.SubAccountID)
			p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).
				With(log.KeyRuntimeID, runtime.RuntimeID).Debug("Deleted subAccount from cache")
			// delete metrics for old shoot name.
			if record, ok := recordObj.(kmccache.Record); ok {
				if success := deleteMetrics(record); !success {
					p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyRuntimeID, runtime.RuntimeID).
						Info("prometheus metrics were not successfully removed for subAccount")
				}
			}
			continue
		}
		p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).
			With(log.KeyRuntimeID, runtime.RuntimeID).Debug("Ignoring SubAccount as it is not trackable")
	}

	// Cleaning up subAccounts from the cache which are not returned by KEB anymore
	for sAccID, recordObj := range p.Cache.Items() {
		if _, ok := validSubAccounts[sAccID]; !ok {
			record, ok := recordObj.Object.(kmccache.Record)
			p.Cache.Delete(sAccID)
			if !ok {
				p.namedLogger().With(log.KeySubAccountID, sAccID).
					Error("bad item from cache, could not cast to a record obj")
			} else {
				p.namedLogger().With(log.KeySubAccountID, sAccID).With(log.KeyRuntimeID, record.RuntimeID).
					Debug("SubAccount is not trackable anymore hence deleting it from cache")
			}
			// delete metrics for old shoot name.
			if success := deleteMetrics(record); !success {
				p.namedLogger().With(log.KeySubAccountID, record.SubAccountID).With(log.KeyRuntimeID, record.RuntimeID).
					Info("prometheus metrics were not successfully removed for subAccount")
			}
		}
	}
}

func (p *Process) getSubAccountFromCache(subAccountID string) *kmccache.Record {
	obj, found := p.Cache.Get(subAccountID)
	if found {
		if record, ok := obj.(kmccache.Record); ok {
			return &record
		}
	}
	return nil
}

func (p *Process) namedLogger() *zap.SugaredLogger {
	return p.Logger.With("component", "kmc")
}

func (p *Process) namedLoggerWithRuntime(record *kmccache.Record) *zap.SugaredLogger {
	if record == nil {
		return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, "")
	}
	return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, record.RuntimeID)
}
