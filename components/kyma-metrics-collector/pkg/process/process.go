package process

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	gardenersecret "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/secret"
	gardenershoot "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/shoot"
	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"
	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/util/workqueue"

	"github.com/pkg/errors"

	kebruntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	"github.com/patrickmn/go-cache"
)

type Process struct {
	KEBClient       *keb.Client
	EDPClient       *edp.Client
	Queue           workqueue.DelayingInterface
	ShootClient     *gardenershoot.Client
	SecretClient    *gardenersecret.Client
	Cache           *cache.Cache
	Providers       *Providers
	ScrapeInterval  time.Duration
	WorkersPoolSize int
	NodeConfig      skrnode.ConfigInf
	PVCConfig       skrpvc.ConfigInf
	SvcConfig       skrsvc.ConfigInf
	Logger          *zap.SugaredLogger
}

const (
	shootKubeconfigKey = "kubeconfig"
)

var (
	errorSubAccountIDNotTrackable = errors.New("subAccountID is not trackable")
)

func (p Process) generateRecordWithNewMetrics(identifier int, subAccountID string) (record kmccache.Record, err error) {
	ctx := context.Background()
	var ok bool

	obj, isFound := p.Cache.Get(subAccountID)
	if !isFound {
		err = errorSubAccountIDNotTrackable
		return
	}

	if record, ok = obj.(kmccache.Record); !ok {
		err = fmt.Errorf("bad item from cache, could not cast to a record obj")
		return
	}
	p.namedLogger().Debugf("[worker: %d] record found from cache: %+v", identifier, record)

	shootName := record.ShootName

	if record.KubeConfig == "" {
		// Get shoot kubeconfig secret
		var secret *corev1.Secret
		secret, err = p.SecretClient.Get(ctx, shootName)
		if err != nil {
			return
		}

		record.KubeConfig = string(secret.Data[shootKubeconfigKey])
		if record.KubeConfig == "" {
			err = fmt.Errorf("kubeconfig for shoot not found")
			return
		}
	}

	// Get shoot CR
	var shoot *gardenerv1beta1.Shoot
	shoot, err = p.ShootClient.Get(ctx, shootName)
	if err != nil {
		return
	}

	// Get nodes dynamic client
	nodesClient, err := p.NodeConfig.NewClient(record.KubeConfig)
	if err != nil {
		return
	}

	// Get nodes
	var nodes *corev1.NodeList
	nodes, err = nodesClient.List(ctx)
	if err != nil {
		return
	}

	if len(nodes.Items) == 0 {
		err = fmt.Errorf("no nodes to process")
		return
	}

	// Get PVCs
	pvcClient, err := p.PVCConfig.NewClient(record.KubeConfig)
	if err != nil {
		return
	}
	var pvcList *corev1.PersistentVolumeClaimList
	pvcList, err = pvcClient.List(ctx)
	if err != nil {
		return
	}

	// Get Svcs
	var svcList *corev1.ServiceList
	svcClient, err := p.SvcConfig.NewClient(record.KubeConfig)
	if err != nil {
		return
	}
	svcList, err = svcClient.List(ctx)
	if err != nil {
		return
	}

	// Create input
	input := Input{
		shoot:    shoot,
		nodeList: nodes,
		pvcList:  pvcList,
		svcList:  svcList,
	}
	metric, err := input.Parse(p.Providers)
	record.Metric = metric
	return
}

// getOldRecordIfMetricExists gets old record from cache if old metric exists
func (p Process) getOldRecordIfMetricExists(subAccountID string) (*kmccache.Record, error) {
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
	p.Logger.Error(notFoundErr)
	return nil, notFoundErr
}

// pollKEBForRuntimes polls KEB for runtimes information
func (p *Process) pollKEBForRuntimes() {
	kebReq, err := p.KEBClient.NewRequest()

	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
			Fatal("create a new request for KEB")
	}
	for {
		runtimesPage, err := p.KEBClient.GetAllRuntimes(kebReq)
		if err != nil {
			p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
				Error("get runtimes from KEB")
			time.Sleep(p.KEBClient.Config.PollWaitDuration)
			continue
		}
		clustersScraped.WithLabelValues(kebReq.RequestURI).Set(float64(runtimesPage.Count))

		p.namedLogger().Debugf("num of runtimes are: %d", runtimesPage.Count)
		p.populateCacheAndQueue(runtimesPage)
		p.namedLogger().Debugf("length of the cache after KEB is done populating: %d", p.Cache.ItemCount())
		p.namedLogger().Infof("waiting to poll KEB again after %v....", p.KEBClient.Config.PollWaitDuration)
		time.Sleep(p.KEBClient.Config.PollWaitDuration)
	}
}

// Start runs the complete process of collection and sending metrics
func (p Process) Start() {

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

// Execute is executed by each worker to process an entry from the queue
func (p *Process) execute(identifier int) {

	for {

		// Pick up a subAccountID to process from queue and mark as Done()
		subAccountIDObj, _ := p.Queue.Get()
		subAccountID := fmt.Sprintf("%v", subAccountIDObj)

		// TODO Implement cleanup holistically in #kyma-project/control-plane/issues/512
		//if isShuttingDown {
		//	//p.Cleanup()
		//	return
		//}

		p.processSubAccountID(subAccountID, identifier)
		p.Queue.Done(subAccountIDObj)
	}
}

func (p Process) processSubAccountID(subAccountID string, identifier int) {
	var payload []byte
	if strings.TrimSpace(subAccountID) == "" {
		p.namedLogger().Warnf("[worker: %d] cannot work with empty subAccountID", identifier)

		// Nothing to do further
		return
	}
	p.namedLogger().Debugf("[worker: %d] subaccid: %v is fetched from queue", identifier, subAccountID)

	record, isOldMetricValid, err := p.getRecordWithOldOrNewMetric(identifier, subAccountID)
	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
			Errorf("[worker: %d] no metric found/generated for subaccount id", identifier)
		// SubAccountID is not trackable anymore as there is no runtime
		if errors.Is(err, errorSubAccountIDNotTrackable) {
			p.namedLogger().With(log.KeyRequeue, log.ValueFalse).
				Infof("[worker: %d] is not requeued subAccountID %s", identifier, subAccountID)
			return
		}
		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLogger().With(log.KeyRequeue, log.ValueTrue).
			Debugf("[worker: %d] successfully requeued after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

		// Nothing to do further
		return
	}

	// Convert metric to JSON
	payload, err = json.Marshal(*record.Metric)
	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
			Errorf("[worker: %d] json.Marshal metric for subaccount id: %s", identifier, subAccountID)

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLogger().With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
			Debugf("[worker: %d] requeued after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

		// Nothing to do further
		return
	}

	// Send metrics to EDP
	// Note: EDP refers SubAccountID as tenant
	p.namedLogger().Debugf("[worker: %d] sending EventStreamToEDP: tenant: %s payload: %s", identifier, subAccountID, string(payload))
	err = p.sendEventStreamToEDP(subAccountID, payload)
	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
			Errorf("[worker: %d] send metric to EDP for subAccountID: %s, event-stream: %s", identifier, subAccountID, string(payload))

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLogger().With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
			Debugf("[worker: %d] requeued after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

		// Nothing to do further hence continue
		return
	}
	p.namedLogger().With(log.KeyResult, log.ValueSuccess).
		Infof("[worker: %d] sent event stream for subaccountID: %s, shoot: %s", identifier, subAccountID, record.ShootName)

	if !isOldMetricValid {
		p.Cache.Set(record.SubAccountID, *record, cache.NoExpiration)
		p.namedLogger().With(log.KeyResult, log.ValueSuccess).
			Debugf("[worker: %d] saved metric for subAccountID %s", identifier, record.SubAccountID)
	}

	// Requeue the subAccountID anyway
	p.namedLogger().With(log.KeyResult, log.ValueSuccess).With(log.KeyRequeue, log.ValueTrue).
		Debugf("[worker: %d] requeued after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)
	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
}

// getRecordWithOldOrNewMetric generates new metric or fetches the old metric along with a bool flag which
// indicates whether it is an old metric or not(true, when it is old and false when it is new)
func (p Process) getRecordWithOldOrNewMetric(identifier int, subAccountID string) (*kmccache.Record, bool, error) {
	record, err := p.generateRecordWithNewMetrics(identifier, subAccountID)
	if err != nil {
		if errors.Is(err, errorSubAccountIDNotTrackable) {
			p.namedLogger().Infof("[worker: %d] subAccountID: %s is not trackable anymore, skipping the fetch of old metric", identifier, subAccountID)
			return nil, false, err
		}
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
			Errorf("generate new metric for subaccountID: %v", subAccountID)
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

func (p Process) sendEventStreamToEDP(tenant string, payload []byte) error {
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

func isClusterTrackable(runtime *kebruntime.RuntimeDTO) bool {
	if runtime.Status.Provisioning != nil &&
		runtime.Status.Provisioning.State == "succeeded" &&
		runtime.Status.Deprovisioning == nil {
		return true
	}
	return false
}

// populateCacheAndQueue populates Cache and Queue with new runtimes and deletes the runtimes which should not be tracked
func (p *Process) populateCacheAndQueue(runtimes *kebruntime.RuntimesPage) {

	validSubAccounts := make(map[string]bool)
	for _, runtime := range runtimes.Data {
		if runtime.SubAccountID == "" {
			continue
		}
		validSubAccounts[runtime.SubAccountID] = true
		recordObj, isFoundInCache := p.Cache.Get(runtime.SubAccountID)
		if isClusterTrackable(&runtime) {
			newRecord := kmccache.Record{
				SubAccountID: runtime.SubAccountID,
				ShootName:    runtime.ShootName,
				KubeConfig:   "",
				Metric:       nil,
			}
			if !isFoundInCache {
				err := p.Cache.Add(runtime.SubAccountID, newRecord, cache.NoExpiration)
				if err != nil {
					p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err).
						Errorf("add subAccountID: %v to cache hence skipping queueing it", runtime.SubAccountID)
					continue
				}
				p.Queue.Add(runtime.SubAccountID)
				p.namedLogger().With(log.KeyResult, log.ValueSuccess).Debugf("Queued and added to cache: %v", runtime.SubAccountID)
				continue
			}

			// Cluster is trackable and exists in the cache
			if record, ok := recordObj.(kmccache.Record); ok {
				if record.ShootName != runtime.ShootName {
					// The shootname has changed hence the record in the cache is not valid anymore
					// No need to queue as the subAccountID already exists in queue
					p.Cache.Set(runtime.SubAccountID, newRecord, cache.NoExpiration)
					p.namedLogger().Debugf("Resetted the values in cache: %v", runtime.SubAccountID)
				}
			}
			continue
		}
		if isFoundInCache {
			// Cluster is not trackable but is found in cache should be deleted
			p.Cache.Delete(runtime.SubAccountID)
			p.namedLogger().Debugf("Deleted subAccountID: %v from cache", runtime.SubAccountID)
			continue
		}
		p.namedLogger().Debugf("Ignoring SubAccountID: %v, as it is not trackable", runtime.SubAccountID)
	}

	// Cleaning up subAccounts from the cache which are not returned by KEB anymore
	for sAccID := range p.Cache.Items() {
		if _, ok := validSubAccounts[sAccID]; !ok {
			p.Cache.Delete(sAccID)
			p.namedLogger().Debugf("SubAccountID: %v is not trackable anymore hence deleting it from cache", sAccID)
		}
	}
}

func (p *Process) namedLogger() *zap.SugaredLogger {
	return p.Logger.With("component", "kmc-server")
}
