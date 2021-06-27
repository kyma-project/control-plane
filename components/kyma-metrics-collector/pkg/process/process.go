package process

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	gardenersecret "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/secret"
	gardenershoot "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/shoot"
	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"
	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/util/workqueue"

	"github.com/pkg/errors"

	kebruntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
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
	Logger          *logrus.Logger
}

const shootKubeconfigKey = "kubeconfig"

func (p Process) generateRecordWithMetrics(identifier int, subAccountID string) (record kmccache.Record, err error) {
	ctx := context.Background()
	var ok bool

	obj, isFound := p.Cache.Get(subAccountID)
	if !isFound {
		err = fmt.Errorf("subAccountID was not found in cache")
		return
	}

	defer p.Queue.Done(subAccountID)
	if record, ok = obj.(kmccache.Record); !ok {
		err = fmt.Errorf("bad item from cache, could not cast to a record obj")
		return
	}
	p.Logger.Debugf("[worker: %d] record found from cache: %+v", identifier, record)

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
		p.Logger.Fatalf("failed to create a new request for KEB: %v", err)
	}
	for {
		runtimesPage, err := p.KEBClient.GetAllRuntimes(kebReq)
		if err != nil {
			p.Logger.Errorf("failed to get runtimes from KEB: %v", err)
			time.Sleep(p.KEBClient.Config.PollWaitDuration)
			continue
		}
		clustersScraped.WithLabelValues(kebReq.RequestURI).Set(float64(runtimesPage.Count))

		p.Logger.Debugf("num of runtimes are: %d", runtimesPage.Count)
		p.populateCacheAndQueue(runtimesPage)
		p.Logger.Debugf("length of the cache after KEB is done populating: %d", p.Cache.ItemCount())
		p.Logger.Infof("waiting to poll KEB again after %v....", p.KEBClient.Config.PollWaitDuration)
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
			p.Logger.Infof("########  Worker exits ########")
		}()
	}
	wg.Wait()
}

// Execute is executed by each worker to process an entry from the queue
func (p *Process) execute(identifier int) {

	for {
		var payload []byte
		// Pick up a subAccountID to process from queue
		subAccountIDObj, _ := p.Queue.Get()
		// TODO Implement cleanup holistically in #kyma-project/control-plane/issues/512
		//if isShuttingDown {
		//	//p.Cleanup()
		//	return
		//}
		subAccountID := fmt.Sprintf("%v", subAccountIDObj)
		if strings.TrimSpace(subAccountID) == "" {
			p.Logger.Warnf("[worker: %d] cannot work with empty subAccountID", identifier)

			// Nothing to do further
			continue
		}
		p.Logger.Debugf("[worker: %d] subaccid: %v is fetched from queue", identifier, subAccountIDObj)

		record, isOldMetricValid, err := p.getRecordWithOldOrNewMetric(identifier, subAccountID)
		if err != nil {
			p.Logger.Errorf("[worker: %d] no metric found/generated for subaccount id: %v", identifier, err)

			p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
			p.Logger.Debugf("[worker: %d] successfully requed after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

			// Nothing to do further
			continue
		}

		// Convert metric to JSON
		payload, err = json.Marshal(*record.Metric)
		if err != nil {
			p.Logger.Errorf("[worker: %d] failed to json.Marshal metric for subaccount id: %v", identifier, err)

			p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
			p.Logger.Debugf("[worker: %d] successfully requed after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

			// Nothing to do further
			continue
		}

		// Send metrics to EDP
		// Note: EDP refers SubAccountID as tenant
		p.Logger.Debugf("[worker: %d] sending EventStreamToEDP: tenant: %s payload: %s", identifier, subAccountID, string(payload))
		err = p.sendEventStreamToEDP(subAccountID, payload)
		if err != nil {
			p.Logger.Errorf("[worker: %d] failed to send metric to EDP for subAccountID: %s, event-stream: %s, with err: %v", identifier, subAccountID, string(payload), err)

			p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
			p.Logger.Debugf("[worker: %d] successfully requed after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)

			// Nothing to do further hence continue
			continue
		}
		p.Logger.Infof("[worker: %d] successfully sent event stream for subaccountID: %s, shoot: %s", identifier, subAccountID, record.ShootName)

		if !isOldMetricValid {
			p.Cache.Set(record.SubAccountID, *record, cache.NoExpiration)
			p.Logger.Debugf("[worker: %d] successfully saved metric for subAccountID %s", identifier, record.SubAccountID)
		}

		// Requeue the subAccountID anyway
		p.Logger.Debugf("[worker: %d] successfully requed after %v for subAccountID %s", identifier, p.ScrapeInterval, subAccountID)
		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
	}
}

func (p Process) getRecordWithOldOrNewMetric(identifier int, subAccountID string) (*kmccache.Record, bool, error) {
	record, err := p.generateRecordWithMetrics(identifier, subAccountID)
	if err != nil {
		p.Logger.Errorf("failed to generate new metric for subaccountID: %v, err: %v", subAccountID, err)
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

	for _, runtime := range runtimes.Data {
		if runtime.SubAccountID == "" {
			continue
		}
		recordObj, isFound := p.Cache.Get(runtime.SubAccountID)
		if isClusterTrackable(&runtime) {
			newRecord := kmccache.Record{
				SubAccountID: runtime.SubAccountID,
				ShootName:    runtime.ShootName,
				KubeConfig:   "",
				Metric:       nil,
			}
			if !isFound {
				err := p.Cache.Add(runtime.SubAccountID, newRecord, cache.NoExpiration)
				if err != nil {
					p.Logger.Errorf("failed to add subAccountID: %v to cache hence skipping queueing it", err)
					continue
				}
				p.Queue.Add(runtime.SubAccountID)
				p.Logger.Debugf("Queued and added to cache: %v", runtime.SubAccountID)
				continue
			}

			// Cluster is trackable and exists in the cache
			if record, ok := recordObj.(kmccache.Record); ok {
				if record.ShootName != runtime.ShootName {
					// The shootname has changed hence the record in the cache is not valid anymore
					// No need to queue as the subAccountID already exists in queue
					p.Cache.Set(runtime.SubAccountID, newRecord, cache.NoExpiration)
					p.Logger.Debugf("Resetted the values in cache: %v", runtime.SubAccountID)
				}
			}
		} else {
			if isFound {
				p.Cache.Delete(runtime.SubAccountID)
				p.Logger.Debugf("Deleted subAccountID: %v", runtime.SubAccountID)
			}
		}
	}
}
