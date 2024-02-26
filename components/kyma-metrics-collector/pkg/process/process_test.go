package process

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"k8s.io/client-go/kubernetes/fake"

	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"
	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"
	kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	kmckeb "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/onsi/gomega"
	gocache "github.com/patrickmn/go-cache"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/util/workqueue"
)

const (
	// General.
	timeout    = 5 * time.Second
	bigTimeout = 10 * time.Second

	// KEB related variables.
	kebRuntimeResponseFilePath = "../testing/fixtures/runtimes_response_process.json"
	expectedPathPrefix         = "/runtimes"

	// EDP related variables.

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
		kebFetchedClusters.Reset()

		go func() {
			newProcess.pollKEBForRuntimes()
		}()
		g.Eventually(func() int {
			return timesVisited
		}, 10*time.Second).Should(gomega.Equal(expectedTimesVisited))

		// Ensure metric exists
		metricName := "kmc_process_fetched_clusters"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		// check each metric with labels has the expected value
		for _, runtimeData := range expectedRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})
}

func TestIsProvisionedStatus(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// const used in all test cases
	subAccountID := "c7db696a-32fa-48ee-9009-aa3e0034121e"
	shootName := "shoot-gKtxg"

	// test cases
	testCases := []struct {
		name         string
		givenRuntime kebruntime.RuntimeDTO
		expectedBool bool
	}{
		{
			name:         "should return true when runtime is in provisioning state succeeded and provisioning status is not nil and deprovisioning is nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return false when runtime is in provisioning state succeeded and deprovisioning is not nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateSucceeded)),
			expectedBool: false,
		},
		{
			name:         "should return false when runtime is in provisioning state failed and provisioning status is not nil and deprovisioning is nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningFailedState),
			expectedBool: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// when
			isProvisioned := isProvisionedStatus(tc.givenRuntime)

			// then
			g.Expect(isProvisioned).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestIsTrackableState(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// test cases
	testCases := []struct {
		name              string
		givenRuntimeState kebruntime.State
		expectedBool      bool
	}{
		{
			name:              "should return true when shoot is in succeeded state",
			givenRuntimeState: kebruntime.StateSucceeded,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in error state",
			givenRuntimeState: kebruntime.StateError,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in upgrading state",
			givenRuntimeState: kebruntime.StateUpgrading,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in updating state",
			givenRuntimeState: kebruntime.StateUpdating,
			expectedBool:      true,
		},
		{
			name:              "should return false when shoot is in deprovisioned state",
			givenRuntimeState: kebruntime.StateDeprovisioned,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioned incomplete state",
			givenRuntimeState: kebruntime.StateDeprovisionIncomplete,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioning  state",
			givenRuntimeState: kebruntime.StateDeprovisioning,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in failed state",
			givenRuntimeState: kebruntime.StateFailed,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in suspended state",
			givenRuntimeState: kebruntime.StateSuspended,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in provisioning state",
			givenRuntimeState: kebruntime.StateProvisioning,
			expectedBool:      false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// when
			isTrackable := isTrackableState(tc.givenRuntimeState)

			// then
			g.Expect(isTrackable).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestIsRuntimeTrackable(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// const used in all test cases
	subAccountID := "c7db696a-32fa-48ee-9009-aa3e0034121e"
	shootName := "shoot-gKtxg"

	// test cases
	testCases := []struct {
		name         string
		givenRuntime kebruntime.RuntimeDTO
		expectedBool bool
	}{
		{
			name:         "should return true when runtime is in trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return true when runtime is in trackable state and deprovisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return true when runtime is in non-trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateDeprovisioning)),
			expectedBool: true,
		},
		{
			name:         "should return false when runtime is in non-trackable state and deprovisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioning)),
			expectedBool: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// when
			isRuntimeTrackable := isRuntimeTrackable(tc.givenRuntime)

			// then
			g.Expect(isRuntimeTrackable).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestPopulateCacheAndQueue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("runtimes with only provisioned status and other statuses with failures", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

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
			runtime := kmctesting.NewRuntimesDTO(failedID, shootName, kmctesting.WithProvisioningFailedState)
			runtimesPage.Data = append(runtimesPage.Data, runtime)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_process_fetched_clusters"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("runtimes with both provisioned and deprovisioned status", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

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
			rntme := kmctesting.NewRuntimesDTO(failedID, fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)), kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
			runtimesPage.Data = append(runtimesPage.Data, rntme)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := "kmc_process_fetched_clusters"
		numberOfAllClusters := 4
		expectedMetricValue := 1
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache followed by deprovisioning completely(with empty runtimes in KEB response)", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

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
		metricName := "kmc_process_fetched_clusters"
		numberOfAllClusters := 0
		expectedMetricValue := 0
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		for _, runtimeData := range runtimesPageWithNoRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache, then shoot is deprovisioned and provisioned again", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

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

		rntme := kmctesting.NewRuntimesDTO(subAccID, oldShootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
		runtimesPage.Data = append(runtimesPage.Data, rntme)

		// expected cache changes after deprovisioning
		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// provision a new SKR again with a new name
		skrRuntimesPageWithProvisioning := new(kebruntime.RuntimesPage)
		skrRuntimesPageWithProvisioning.Data = []kebruntime.RuntimeDTO{
			kmctesting.NewRuntimesDTO(subAccID, newShootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
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
		metricName := "kmc_process_fetched_clusters"
		// expecting number of all clusters to be 1, as deprovisioned shoot is removed
		// only counting the new shoot
		numberOfAllClusters := 1
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
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

// TestPrometheusMetricsRemovedForDeletedSubAccounts tests that the prometheus metrics
// are deleted by `populateCacheAndQueue` method. It will test the following cases:
// case 1: Cache entry exists for a shoot, but it is not returned by KEB anymore.
// case 2: Shoot with de-provisioned status returned by KEB.
// case 3: Shoot name of existing subAccount changed and cache entry exists with old shoot name.
func TestPrometheusMetricsRemovedForDeletedSubAccounts(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// test cases. These cases are not safe to be run in parallel.
	testCases := []struct {
		name                       string
		givenShoot1                kmccache.Record
		givenShoot2                kmccache.Record
		givenShoot2NewName         string
		givenIsShoot2ReturnedByKEB bool
	}{
		{
			name: "should have removed metrics when cache entry exists for a shoot, but it is not returned by KEB anymore",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenIsShoot2ReturnedByKEB: false,
		},
		{
			name: "should have removed metrics when cache entry exists for a shoot, but KEB returns shoot with de-provisioned status",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenIsShoot2ReturnedByKEB: true,
		},
		{
			name: "should have removed metrics when cache entry exists for a shoot, but KEB returns shoot with different shoot name",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
			},
			givenIsShoot2ReturnedByKEB: true,
			givenShoot2NewName:         fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			// reset metrics.
			subAccountProcessed.Reset()
			subAccountProcessedTimeStamp.Reset()
			oldMetricsPublishedGauge.Reset()

			// add metrics for both shoots.
			recordSubAccountProcessed(false, tc.givenShoot1)
			recordSubAccountProcessed(false, tc.givenShoot2)
			recordOldMetricsPublishedGauge(tc.givenShoot1)
			recordOldMetricsPublishedGauge(tc.givenShoot2)
			recordSubAccountProcessedTimeStamp(false, tc.givenShoot1)
			recordSubAccountProcessedTimeStamp(false, tc.givenShoot2)

			// setup cache
			cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
			// add both shoots to cache
			err := cache.Add(tc.givenShoot1.SubAccountID, tc.givenShoot1, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())
			// define target shoot to test.
			err = cache.Add(tc.givenShoot2.SubAccountID, tc.givenShoot2, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			// init queue.
			queue := workqueue.NewDelayingQueue()
			expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
			err = expectedCache.Add(tc.givenShoot1.SubAccountID, tc.givenShoot1, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			// mock KEB response.
			runtimesPage := new(kebruntime.RuntimesPage)
			runtime := kmctesting.NewRuntimesDTO(tc.givenShoot1.SubAccountID,
				tc.givenShoot1.ShootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
			runtimesPage.Data = append(runtimesPage.Data, runtime)
			if tc.givenIsShoot2ReturnedByKEB {
				runtime = kmctesting.NewRuntimesDTO(tc.givenShoot2.SubAccountID,
					tc.givenShoot2.ShootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
				if tc.givenShoot2NewName != "" {
					runtime = kmctesting.NewRuntimesDTO(tc.givenShoot2.SubAccountID,
						tc.givenShoot2NewName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
				}
				runtimesPage.Data = append(runtimesPage.Data, runtime)
			}

			p := Process{
				Queue:  queue,
				Cache:  cache,
				Logger: logger.NewLogger(zapcore.InfoLevel),
			}

			// when
			p.populateCacheAndQueue(runtimesPage)

			// then
			// check if metrics for existingShoot still exists or not.
			// metric: subAccountProcessed
			gotMetrics, err := subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

			// metric: oldMetricsPublishedGauge
			gotMetrics, err = oldMetricsPublishedGauge.GetMetricWithLabelValues(
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

			// check if metrics for de-provisioned shoot were deleted or not.
			// metric: subAccountProcessed
			gotMetrics, err = subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))

			// metric: oldMetricsPublishedGauge
			gotMetrics, err = oldMetricsPublishedGauge.GetMetricWithLabelValues(
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
		})
	}
}

// TestPrometheusMetricsProcessSubAccountID tests the prometheus metrics maintained by `processSubAccountID` method.
func TestPrometheusMetricsProcessSubAccountID(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// given (common for all test cases).
	logger := logger.NewLogger(zapcore.DebugLevel)
	const givenMethodRecalls = 3
	givenKubeConfig := "eyJmb28iOiAiYmFyIn0="

	// cloud providers.
	providersData, err := kmctesting.LoadFixtureFromFile(providersFile)
	g.Expect(err).Should(gomega.BeNil())
	config := &env.Config{PublicCloudSpecs: string(providersData)}
	givenProviders, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	// setup EDP server.
	edpAllowedSubAccountID := uuid.New().String()
	edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		params := mux.Vars(req)
		if params["tenantID"] == edpAllowedSubAccountID {
			rw.WriteHeader(http.StatusCreated)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
	})
	edpPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/{tenantID}/%s/events", testNamespace,
		testDataStream, testDataStreamVersion, testEnv)
	srv := kmctesting.StartTestServer(edpPath, edpTestHandler, g)
	defer srv.Close()

	// EDP client.
	edpConfig := newEDPConfig(srv.URL)
	edpClient := edp.NewClient(edpConfig, logger)

	// test cases. These cases are not safe to be run in parallel.
	testCases := []struct {
		name                string
		givenShoot          kmccache.Record
		wantSuccess         bool
		wantOldMetricReused bool
	}{
		{
			name: "should have correct metrics when it successfully processes subAccount with new data",
			givenShoot: kmccache.Record{
				SubAccountID:    edpAllowedSubAccountID,
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
				KubeConfig:      givenKubeConfig,
			},
			wantSuccess:         true,
			wantOldMetricReused: false,
		},
		{
			name: "should have correct metrics when it processes subAccount with old data",
			// the method (which is being tested) will use old data,
			// when it fails to query to k8s cluster for information.
			givenShoot: kmccache.Record{
				SubAccountID:    edpAllowedSubAccountID,
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
				Metric:          NewMetric(),
				KubeConfig:      "invalid",
			},
			wantSuccess:         true,
			wantOldMetricReused: true,
		},
		{
			name: "should have correct metrics when it fails to publish data to EDP",
			givenShoot: kmccache.Record{
				SubAccountID:    uuid.New().String(), // not allowed subAccountID in mocked EDP server.
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
				KubeConfig:      givenKubeConfig,
			},
			wantSuccess:         false,
			wantOldMetricReused: false,
		},
		{
			name: "should have correct metrics when it fails to process subAccount",
			// the method (which is being tested) will use old data,
			// when it fails to query to k8s cluster for information and
			// the old data in cache is invalid (e.g. `Metric: nil`).
			givenShoot: kmccache.Record{
				SubAccountID:    edpAllowedSubAccountID,
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    Azure,
				KubeConfig:      "invalid",
				Metric:          nil,
			},
			wantSuccess:         false,
			wantOldMetricReused: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			testStartTimeUnix := time.Now().Unix()
			subAccountProcessed.Reset()
			subAccountProcessedTimeStamp.Reset()
			oldMetricsPublishedGauge.Reset()

			// populate cache.
			cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
			err = cache.Add(tc.givenShoot.SubAccountID, tc.givenShoot, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			// k8s fake clients.
			g.Expect(err).Should(gomega.BeNil())
			secretKCPStored := kmctesting.NewKCPStoredSecret(tc.givenShoot.RuntimeID, tc.givenShoot.KubeConfig)
			secretCacheClient := fake.NewSimpleClientset(secretKCPStored)
			fakeNodeClient := skrnode.FakeNodeClient{}
			fakePVCClient := skrpvc.FakePVCClient{}
			fakeSvcClient := skrsvc.FakeSvcClient{}

			// initiate process instance.
			givenProcess := &Process{
				EDPClient:         edpClient,
				Queue:             workqueue.NewDelayingQueue(),
				SecretCacheClient: secretCacheClient.CoreV1(),
				Cache:             cache,
				Providers:         givenProviders,
				ScrapeInterval:    3 * time.Second,
				Logger:            logger,
				NodeConfig:        fakeNodeClient,
				PVCConfig:         fakePVCClient,
				SvcConfig:         fakeSvcClient,
			}

			// when
			// calling the method multiple times to generate testable metrics.
			for i := 0; i < givenMethodRecalls; i++ {
				givenProcess.processSubAccountID(tc.givenShoot.SubAccountID, i)
			}

			// then
			// check prometheus metrics.
			// metric: subAccountProcessed
			gotMetrics, err := subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(tc.wantSuccess),
				tc.givenShoot.ShootName,
				tc.givenShoot.InstanceID,
				tc.givenShoot.RuntimeID,
				tc.givenShoot.SubAccountID,
				tc.givenShoot.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			// the metric will be incremented even in case of failure, so that is why
			// it should be equal to the number of time the `processSubAccountID` is called.
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(givenMethodRecalls)))

			// metric: oldMetricsPublishedGauge
			gotMetrics, err = oldMetricsPublishedGauge.GetMetricWithLabelValues(
				tc.givenShoot.ShootName,
				tc.givenShoot.InstanceID,
				tc.givenShoot.RuntimeID,
				tc.givenShoot.SubAccountID,
				tc.givenShoot.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			if tc.wantOldMetricReused {
				// it should have kept increasing to track consecutive number of re-use.
				g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(givenMethodRecalls)))
			} else {
				// the metric will be reset to zero when a subAccount is successfully processed.
				g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
			}

			// metric: subAccountProcessedTimeStamp
			gotMetrics, err = subAccountProcessedTimeStamp.GetMetricWithLabelValues(
				strconv.FormatBool(tc.wantOldMetricReused),
				tc.givenShoot.ShootName,
				tc.givenShoot.InstanceID,
				tc.givenShoot.RuntimeID,
				tc.givenShoot.SubAccountID,
				tc.givenShoot.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			// check if the last published time has correct value.
			// the timestamp will only be updated when the subAccount is successfully processed.
			utcTime := testutil.ToFloat64(gotMetrics)
			isPublishedAfterTestStartTime := int64(utcTime) >= testStartTimeUnix
			g.Expect(isPublishedAfterTestStartTime).Should(
				gomega.Equal(tc.wantSuccess),
				"the last published time should be updated only when a new event is published to EDP.")
		})
	}
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
		ProviderType: Azure,
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
		SecretCacheClient: secretCacheClient.CoreV1(),
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
		runtime := kmctesting.NewRuntimesDTO(successfulID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
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

// Helper function to check the value of the `kmc_process_fetched_clusters` metric using `ToFloat64`
func verifyKEBAllClustersCountMetricValue(expectedValue int, g *gomega.WithT, runtimeData kebruntime.RuntimeDTO) bool {
	return g.Eventually(func() int {

		trackable := isClusterTrackable(&runtimeData)

		counter, err := kebFetchedClusters.GetMetricWithLabelValues(
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
