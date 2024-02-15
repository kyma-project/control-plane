package process

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"k8s.io/client-go/kubernetes/fake"

	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"
	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"

	gardenershoot "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/shoot"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	corev1 "k8s.io/api/core/v1"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/commons"

	gardenersecret "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/secret"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"

	"github.com/google/uuid"

	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	kmckeb "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"

	"github.com/onsi/gomega"

	"go.uber.org/zap/zapcore"

	gocache "github.com/patrickmn/go-cache"
	"k8s.io/client-go/util/workqueue"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
)

const (
	// General
	timeout    = 5 * time.Second
	bigTimeout = 10 * time.Second

	// KEB related variables
	kebRuntimeResponseFilePath = "../testing/fixtures/runtimes_response.json"
	expectedPathPrefix         = "/runtimes"

	// EDP related variables
	//testTenant            = "testTenant"
	testDataStream        = "dataStream"
	testNamespace         = "namespace"
	testDataStreamVersion = "v1"
	testToken             = "token"
	testEnv               = "env"
	retryCount            = 1
)

func TestGetOldRecordIfMetricExists(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
	expectedSubAccIDToExist := uuid.New().String()
	expectedRecord := kmccache.Record{
		SubAccountID: expectedSubAccIDToExist,
		ShootName:    fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
		KubeConfig:   "foo",
		Metric:       NewMetric(),
	}
	expectedSubAccIDWithNoMetrics := uuid.New().String()
	recordsToBeAdded := []kmccache.Record{
		expectedRecord,
		{
			SubAccountID: uuid.New().String(),
			ShootName:    fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
			KubeConfig:   "foo",
		},
		{
			SubAccountID: expectedSubAccIDWithNoMetrics,
			ShootName:    "",
			KubeConfig:   "",
		},
	}
	for _, record := range recordsToBeAdded {
		err := cache.Add(record.SubAccountID, record, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())
	}

	p := Process{
		Cache:  cache,
		Logger: logger.NewLogger(zapcore.InfoLevel),
	}

	t.Run("old metric found for a subAccountID", func(t *testing.T) {
		gotRecord, err := p.getOldRecordIfMetricExists(expectedSubAccIDToExist)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(*gotRecord).To(gomega.Equal(expectedRecord))
	})

	t.Run("old metric not found for a subAccountID", func(t *testing.T) {
		subAccIDWhichDoesNotExist := uuid.New().String()
		_, err := p.getOldRecordIfMetricExists(subAccIDWhichDoesNotExist)
		g.Expect(err).ShouldNot(gomega.BeNil())
		g.Expect(err.Error()).To(gomega.Equal(fmt.Sprintf("subAccountID: %s not found", subAccIDWhichDoesNotExist)))
	})

	t.Run("old metric found for a subAccountID but does not have metric", func(t *testing.T) {
		_, err := p.getOldRecordIfMetricExists(expectedSubAccIDWithNoMetrics)
		g.Expect(err).ShouldNot(gomega.BeNil())
		g.Expect(err.Error()).To(gomega.Equal(fmt.Sprintf("old metrics for subAccountID: %s not found", expectedSubAccIDWithNoMetrics)))
	})
}

func TestPollKEBForRuntimes(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("execute KEB poller for 2 times", func(t *testing.T) {

		runtimesResponse, err := kmctesting.LoadFixtureFromFile(kebRuntimeResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		expectedRuntimes := new(kebruntime.RuntimesPage)
		err = json.Unmarshal(runtimesResponse, expectedRuntimes)
		g.Expect(err).Should(gomega.BeNil())
		timesVisited := 0
		expectedTimesVisited := 2
		var newProcess *Process

		getRuntimesHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			timesVisited += 1
			t.Logf("time visited: %d", timesVisited)
			g.Expect(req.URL.Path).To(gomega.Equal(expectedPathPrefix))
			_, err := rw.Write(runtimesResponse)
			g.Expect(err).Should(gomega.BeNil())
			rw.WriteHeader(http.StatusOK)
		})

		// Start a local test HTTP server
		srv := kmctesting.StartTestServer(expectedPathPrefix, getRuntimesHandler, g)
		defer srv.Close()
		// Wait until test server is ready
		g.Eventually(func() int {
			// Ignoring error is ok as it goes for retry for non-200 cases
			healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
			t.Logf("retrying :%v", err)
			return healthResp.StatusCode
		}, timeout).Should(gomega.Equal(http.StatusOK))

		kebURL := fmt.Sprintf("%s%s", srv.URL, expectedPathPrefix)

		config := &kmckeb.Config{
			URL:              kebURL,
			Timeout:          timeout,
			RetryCount:       1,
			PollWaitDuration: 2 * time.Second,
		}
		kebClient := &kmckeb.Client{
			HTTPClient: http.DefaultClient,
			Logger:     logger.NewLogger(zapcore.InfoLevel),
			Config:     config,
		}

		queue := workqueue.NewDelayingQueue()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		newProcess = &Process{
			KEBClient:      kebClient,
			Queue:          queue,
			Cache:          cache,
			ScrapeInterval: 0,
			Logger:         logger.NewLogger(zapcore.InfoLevel),
		}

		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		go func() {
			newProcess.pollKEBForRuntimes()
		}()
		g.Eventually(func() int {
			return timesVisited
		}, 10*time.Second).Should(gomega.Equal(expectedTimesVisited))

		// Ensure metric exists
		metricName := "kmc_keb_number_clusters_scraped"
		numberOfRuntimes := 4
		g.Eventually(testutil.CollectAndCount(clustersScraped, metricName)).Should(gomega.Equal(1))
		g.Eventually(func() int {
			counter, err := clustersScraped.GetMetricWithLabelValues("")
			g.Expect(err).Should(gomega.BeNil())
			// check the value of the metric
			return int(testutil.ToFloat64(counter))
		}).Should(gomega.Equal(numberOfRuntimes))

		metricName = "kmc_keb_all_clusters_count"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		// check each metric with labels has the expected value
		for _, runtimeData := range expectedRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})
}

func TestPopulateCacheAndQueue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("runtimes with only provisioned status and other statuses with failures", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		provisionedSuccessfullySubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		provisionedFailedSubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewDelayingQueue()
		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		runtimesPage := new(kebruntime.RuntimesPage)

		expectedQueue := workqueue.NewDelayingQueue()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPage, expectedCache, expectedQueue, err := AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage, provisionedSuccessfullySubAccIDs, expectedCache, expectedQueue)
		g.Expect(err).Should(gomega.BeNil())

		for _, failedID := range provisionedFailedSubAccIDs {
			shootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
			runtime := kmctesting.NewRuntimesDTO(failedID, shootName, kmctesting.WithFailedState)
			runtimesPage.Data = append(runtimesPage.Data, runtime)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_keb_all_clusters_count"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("runtimes with both provisioned and deprovisioned status", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		provisionedSuccessfullySubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		provisionedAndDeprovisionedSubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewDelayingQueue()
		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		runtimesPage := new(kebruntime.RuntimesPage)

		expectedQueue := workqueue.NewDelayingQueue()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPage, expectedCache, expectedQueue, err := AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage, provisionedSuccessfullySubAccIDs, expectedCache, expectedQueue)
		g.Expect(err).Should(gomega.BeNil())

		for _, failedID := range provisionedAndDeprovisionedSubAccIDs {
			rntme := kmctesting.NewRuntimesDTO(failedID, fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)), kmctesting.WithProvisionedAndDeprovisionedState)
			runtimesPage.Data = append(runtimesPage.Data, rntme)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_keb_all_clusters_count"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache but shoot name changed", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		subAccID := uuid.New().String()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewDelayingQueue()
		oldShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
		newShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))

		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		oldRecord := NewRecord(subAccID, oldShootName, "foo")
		newRecord := NewRecord(subAccID, newShootName, "")

		err := p.Cache.Add(subAccID, oldRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage := new(kebruntime.RuntimesPage)
		expectedQueue := workqueue.NewDelayingQueue()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		err = expectedCache.Add(subAccID, newRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		rntme := kmctesting.NewRuntimesDTO(subAccID, newShootName, kmctesting.WithSucceededState)
		runtimesPage.Data = append(runtimesPage.Data, rntme)

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_keb_all_clusters_count"
		numberOfAllClusters := 1
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache followed by deprovisioning completely(with empty runtimes in KEB response)", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		subAccID := uuid.New().String()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewDelayingQueue()
		oldShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))

		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		oldRecord := NewRecord(subAccID, oldShootName, "foo")

		err := p.Cache.Add(subAccID, oldRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPageWithNoRuntimes := new(kebruntime.RuntimesPage)
		expectedEmptyQueue := workqueue.NewDelayingQueue()
		expectedEmptyCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPageWithNoRuntimes.Data = []kebruntime.RuntimeDTO{}

		p.populateCacheAndQueue(runtimesPageWithNoRuntimes)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedEmptyCache))
		g.Expect(areQueuesEqual(p.Queue, expectedEmptyQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_keb_all_clusters_count"
		numberOfAllClusters := 0
		expectedMetricValue := 0
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPageWithNoRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache, then shoot is deprovisioned and provisioned again", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebAllClustersCount.Reset()

		subAccID := uuid.New().String()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewDelayingQueue()
		oldShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
		newShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))

		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		oldRecord := NewRecord(subAccID, oldShootName, "foo")

		err := p.Cache.Add(subAccID, oldRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage := new(kebruntime.RuntimesPage)
		expectedQueue := workqueue.NewDelayingQueue()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		rntme := kmctesting.NewRuntimesDTO(subAccID, oldShootName, kmctesting.WithProvisionedAndDeprovisionedState)
		runtimesPage.Data = append(runtimesPage.Data, rntme)

		// expected cache changes after deprovisioning
		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// provision a new SKR again with a new name
		skrRuntimesPageWithProvisioning := new(kebruntime.RuntimesPage)
		skrRuntimesPageWithProvisioning.Data = []kebruntime.RuntimeDTO{
			kmctesting.NewRuntimesDTO(subAccID, newShootName, kmctesting.WithSucceededState),
		}

		// expected cache changes after provisioning
		newRecord := NewRecord(subAccID, newShootName, "")
		err = expectedCache.Add(subAccID, newRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage.Data = []kebruntime.RuntimeDTO{rntme}
		p.populateCacheAndQueue(skrRuntimesPageWithProvisioning)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		gotSubAccID, _ := p.Queue.Get()
		g.Expect(gotSubAccID).To(gomega.Equal(subAccID))

		// Ensure metric exists
		metricName := "kmc_keb_all_clusters_count"
		// expecting number of all clusters to be 1, as deprovisioned shoot is removed
		// only counting the new shoot
		numberOfAllClusters := 1
		g.Eventually(testutil.CollectAndCount(kebAllClustersCount, metricName)).Should(gomega.Equal(numberOfAllClusters))
		// old shoot should not be present in the metric
		for _, runtimeData := range runtimesPage.Data {
			expectedMetricValue := 0
			switch shootName := runtimeData.ShootName; shootName {
			case oldShootName:
				expectedMetricValue = 0
			case newShootName:
				expectedMetricValue = 1
			}

			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})
}

func TestExecute(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	subAccID := uuid.New().String()
	runtimeID := uuid.New().String()
	tenant := subAccID
	expectedKubeconfig := "eyJmb28iOiAiYmFyIn0="
	expectedPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events", testNamespace, testDataStream, testDataStreamVersion, tenant, testEnv)
	log := logger.NewLogger(zapcore.DebugLevel)

	timesVisited := 0
	// Set up EDP Test Server handler
	expectedHeaders := expectedHeadersInEDPReq()
	edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		timesVisited += 1
		g.Expect(req.Header).To(gomega.Equal(expectedHeaders))
		g.Expect(req.URL.Path).To(gomega.Equal(expectedPath))
		g.Expect(req.Method).To(gomega.Equal(http.MethodPost))
		rw.WriteHeader(http.StatusCreated)
	})

	srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, g)
	defer srv.Close()

	edpConfig := newEDPConfig(srv.URL)
	edpClient := edp.NewClient(edpConfig, log)
	shootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
	secretKCPStored := kmctesting.NewKCPStoredSecret(runtimeID, expectedKubeconfig)

	// Populate cache
	cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
	newRecord := kmccache.Record{
		SubAccountID: subAccID,
		RuntimeID:    runtimeID,
		ShootName:    shootName,
		KubeConfig:   "",
		Metric:       nil,
	}
	expectedRecord := newRecord
	expectedRecord.KubeConfig = expectedKubeconfig
	expectedRecord.Metric = NewMetric()
	expectedRecord.Metric.RuntimeId = runtimeID
	expectedRecord.Metric.SubAccountId = subAccID
	expectedRecord.Metric.ShootName = shootName

	err := cache.Add(subAccID, newRecord, gocache.NoExpiration)
	g.Expect(err).Should(gomega.BeNil())

	// Populate queue
	queue := workqueue.NewDelayingQueue()
	queue.Add(subAccID)

	shoot := kmctesting.GetShoot(shootName, kmctesting.WithAzureProviderAndStandardD8V3VMs)
	shootClient, err := NewFakeShootClient(shoot)
	g.Expect(err).Should(gomega.BeNil())
	secretCacheClient := fake.NewSimpleClientset(secretKCPStored)

	providersData, err := kmctesting.LoadFixtureFromFile(providersFile)
	g.Expect(err).Should(gomega.BeNil())
	config := &env.Config{PublicCloudSpecs: string(providersData)}
	providers, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())
	fakeNodeClient := skrnode.FakeNodeClient{}
	fakePVCClient := skrpvc.FakePVCClient{}
	fakeSvcClient := skrsvc.FakeSvcClient{}

	newProcess := &Process{
		EDPClient:         edpClient,
		Queue:             queue,
		ShootClient:       shootClient,
		SecretCacheClient: secretCacheClient,
		Cache:             cache,
		Providers:         providers,
		ScrapeInterval:    3 * time.Second,
		Logger:            log,
		NodeConfig:        fakeNodeClient,
		PVCConfig:         fakePVCClient,
		SvcConfig:         fakeSvcClient,
	}

	go func() {
		newProcess.execute(1)
	}()

	// Test scrape interval
	g.Eventually(func() bool {
		// With a ScrapeInterval of 3 secs in an interval of 10 seconds, expected timesVisited is atleast 2.
		return timesVisited >= 2
	}, bigTimeout).Should(gomega.BeTrue())

	// Test cache state
	g.Eventually(newProcess.Cache.ItemCount(), timeout).Should(gomega.Equal(len(cache.Items())))
	g.Eventually(func() error {
		gotItemFromCache, found := newProcess.Cache.Get(subAccID)
		if !found {
			return fmt.Errorf("subAccID not found in cache")
		}
		record, ok := gotItemFromCache.(kmccache.Record)
		g.Expect(ok).To(gomega.BeTrue())
		if record.KubeConfig != expectedRecord.KubeConfig {
			return fmt.Errorf("kubeconfigs mismatch, got: %s,expected: %s", record.KubeConfig, expectedRecord.KubeConfig)
		}
		if !reflect.DeepEqual(record.Metric.Networking, expectedRecord.Metric.Networking) {
			g.Expect(record.Metric.Networking).To(gomega.Equal(expectedRecord.Metric.Networking))
			return fmt.Errorf("networking data mismatch, got: %v, expected: %v", record.Metric.Networking, expectedRecord.Metric.Networking)
		}
		if !reflect.DeepEqual(record.Metric.Compute, expectedRecord.Metric.Compute) {
			g.Expect(record.Metric.Compute).To(gomega.Equal(expectedRecord.Metric.Compute))
			return fmt.Errorf("compute data mismatch, got: %v, expected: %v", record.Metric.Compute, expectedRecord.Metric.Compute)
		}

		// check if IDs are correct.
		g.Expect(record.Metric.RuntimeId).To(gomega.Equal(expectedRecord.Metric.RuntimeId))
		g.Expect(record.Metric.SubAccountId).To(gomega.Equal(expectedRecord.Metric.SubAccountId))
		g.Expect(record.Metric.ShootName).To(gomega.Equal(expectedRecord.Metric.ShootName))
		return nil
	}, bigTimeout).Should(gomega.BeNil())

	// The process should keep on publishing events for this subaccount to EDP.
	// We confirm this by check if the count of published events is getting increased.
	oldEventsSentCount := timesVisited
	g.Eventually(func() bool {
		// With a ScrapeInterval of 3 secs in an interval of 10 seconds, expected timesVisited should have at least
		// increased by 2.
		return timesVisited >= oldEventsSentCount+2
	}, bigTimeout).Should(gomega.BeTrue())

	// Clean it from the cache once SKR is deprovisioned
	newProcess.Cache.Delete(subAccID)
	go func() {
		newProcess.execute(1)
	}()
	// The process should not publish more events for this subaccount to EDP.
	// We confirm this by check if the count of published events remains the same after some time.
	oldEventsSentCount = timesVisited
	time.Sleep(timeout)
	g.Expect(timesVisited).To(gomega.Equal(oldEventsSentCount))
	// the queue should be empty.
	g.Eventually(newProcess.Queue.Len()).Should(gomega.Equal(0))
}

func NewFakeShootClient(shoot *gardenerv1beta1.Shoot) (*gardenershoot.Client, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(shoot)
	if err != nil {
		return nil, err
	}
	shootUnstructured := &unstructured.Unstructured{Object: unstructuredMap}
	shootUnstructured.SetGroupVersionKind(gardenershoot.GroupVersionKind())

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, shootUnstructured)
	nsResourceClient := dynamicClient.Resource(gardenershoot.GroupVersionResource()).Namespace("default")

	return &gardenershoot.Client{ResourceClient: nsResourceClient}, nil
}

func NewFakeSecretClient(secret *corev1.Secret) (*gardenersecret.Client, error) {
	scheme, err := commons.SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return nil, err
	}
	secretUnstructured := &unstructured.Unstructured{Object: unstructuredMap}
	secretUnstructured.SetGroupVersionKind(gardenersecret.GroupVersionKind())

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, secretUnstructured)
	nsResourceClient := dynamicClient.Resource(gardenersecret.GroupVersionResource()).Namespace("default")

	return &gardenersecret.Client{ResourceClient: nsResourceClient}, nil
}

func NewRecord(subAccId, shootName, kubeconfig string) kmccache.Record {
	return kmccache.Record{
		SubAccountID: subAccId,
		ShootName:    shootName,
		KubeConfig:   kubeconfig,
		Metric:       nil,
	}
}

func areQueuesEqual(src, dest workqueue.DelayingInterface) bool {
	if src.Len() != dest.Len() {
		return false
	}
	for src.Len() > 0 {
		srcItem, _ := src.Get()
		destItem, _ := dest.Get()
		if srcItem != destItem {
			return false
		}
	}
	return true
}

func AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage *kebruntime.RuntimesPage, successfulIDs []string, expectedCache *gocache.Cache, expectedQueue workqueue.DelayingInterface) (*kebruntime.RuntimesPage, *gocache.Cache, workqueue.DelayingInterface, error) {
	for _, successfulID := range successfulIDs {
		shootID := kmctesting.GenerateRandomAlphaString(5)
		shootName := fmt.Sprintf("shoot-%s", shootID)
		runtime := kmctesting.NewRuntimesDTO(successfulID, shootName, kmctesting.WithSucceededState)
		runtimesPage.Data = append(runtimesPage.Data, runtime)
		err := expectedCache.Add(successfulID, kmccache.Record{
			SubAccountID: successfulID,
			ShootName:    shootName,
		}, gocache.NoExpiration)
		if err != nil {
			return nil, nil, nil, err
		}
		expectedQueue.Add(successfulID)
	}
	return runtimesPage, expectedCache, expectedQueue, nil
}

func newEDPConfig(url string) *edp.Config {
	return &edp.Config{
		URL:               url,
		Token:             testToken,
		Namespace:         testNamespace,
		DataStreamName:    testDataStream,
		DataStreamVersion: testDataStreamVersion,
		DataStreamEnv:     testEnv,
		Timeout:           timeout,
		EventRetry:        retryCount,
	}
}

func expectedHeadersInEDPReq() http.Header {
	return http.Header{
		"Authorization":   []string{fmt.Sprintf("Bearer %s", testToken)},
		"Accept-Encoding": []string{"gzip"},
		"User-Agent":      []string{"kyma-metrics-collector"},
		"Content-Type":    []string{"application/json;charset=utf-8"},
	}
}

func NewMetric() *edp.ConsumptionMetrics {
	return &edp.ConsumptionMetrics{
		Timestamp: "",
		Compute: edp.Compute{
			VMTypes: []edp.VMType{
				{
					Name:  "standard_d8_v3",
					Count: 3,
				},
			},
			ProvisionedCpus:  24,
			ProvisionedRAMGb: 96,
			ProvisionedVolumes: edp.ProvisionedVolumes{
				SizeGbTotal:   30,
				Count:         2,
				SizeGbRounded: 64,
			},
		},
		Networking: edp.Networking{
			ProvisionedVnets: 1,
			ProvisionedIPs:   2,
		},
	}
}

// Helper function to check if a cluster is trackable
func isClusterTrackable(runtime *kebruntime.RuntimeDTO) bool {
	// Check if the cluster is in a trackable state
	trackableStates := map[kebruntime.State]bool{
		"succeeded": true,
		"error":     true,
		"upgrading": true,
		"updating":  true,
	}

	if trackableStates[runtime.Status.State] ||
		(runtime.Status.Provisioning != nil &&
			runtime.Status.Provisioning.State == "succeeded" &&
			runtime.Status.Deprovisioning == nil) {
		return true
	}
	return false
}

// Helper function to check the value of the `kmc_keb_all_clusters_count` metric using `ToFloat64`
func verifyKEBAllClustersCountMetricValue(expectedValue int, g *gomega.WithT, runtimeData kebruntime.RuntimeDTO) bool {
	return g.Eventually(func() int {

		trackable := isClusterTrackable(&runtimeData)

		counter, err := kebAllClustersCount.GetMetricWithLabelValues(
			strconv.FormatBool(trackable),
			runtimeData.ShootName,
			runtimeData.InstanceID,
			runtimeData.RuntimeID,
			runtimeData.SubAccountID,
			runtimeData.GlobalAccountID)

		g.Expect(err).Should(gomega.BeNil())
		// check the value of the metric
		return int(testutil.ToFloat64(counter))
	}).Should(gomega.Equal(expectedValue))
}

// Helper function to check the occurrence of the `kmc_keb_all_clusters_count` metric using `CollectAndCount`
func verifyKEBAllClustersCountMetricOccurrence(metricName string, expectedOccurrence int, g *gomega.WithT, runtimeData kebruntime.RuntimeDTO) bool {
	return g.Eventually(func() int {

		counter, err := kebAllClustersCount.GetMetricWithLabelValues(
			"",
			runtimeData.ShootName,
			runtimeData.InstanceID,
			runtimeData.RuntimeID,
			runtimeData.SubAccountID,
			runtimeData.GlobalAccountID)

		g.Expect(err).Should(gomega.BeNil())
		// check the occurrence of the metric with above labels
		return testutil.CollectAndCount(counter, metricName)
	}).Should(gomega.Equal(expectedOccurrence))
}
