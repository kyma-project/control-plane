package keb

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
)

const (
	timeout                         = 5 * time.Second
	expectedPathPrefix              = "/runtimes"
	expectedPathPrefixWith1Page     = "/runtimes/with1page"
	kebRuntimeResponseFilePath      = "../testing/fixtures/runtimes_response.json"
	kebRuntimePage1ResponseFilePath = "../testing/fixtures/runtimes_response_page1.json"
	kebRuntimePage2ResponseFilePath = "../testing/fixtures/runtimes_response_page2.json"

	// Metrics related variables
	metricsName   = "kmc_keb_request_total"
	histogramName = "kmc_keb_request_duration_seconds"
)

func TestGetAllRuntimes(t *testing.T) {
	t.Run("when 2 pages are returned for all runtimes on matching path and HTTP 404 for non matched ones", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		// given
		// reset metrics state.
		latencyMetric.Reset()

		runtimesResponse, err := kmctesting.LoadFixtureFromFile(kebRuntimeResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage1Response, err := kmctesting.LoadFixtureFromFile(kebRuntimePage1ResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage2Response, err := kmctesting.LoadFixtureFromFile(kebRuntimePage2ResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		expectedRuntimes := new(runtime.RuntimesPage)
		err = json.Unmarshal(runtimesResponse, expectedRuntimes)
		g.Expect(err).Should(gomega.BeNil())

		expectedPage1Runtimes := new(runtime.RuntimesPage)
		err = json.Unmarshal(runtimesPage1Response, expectedPage1Runtimes)
		g.Expect(err).Should(gomega.BeNil())

		expectedPage2Runtimes := new(runtime.RuntimesPage)
		err = json.Unmarshal(runtimesPage2Response, expectedPage2Runtimes)
		g.Expect(err).Should(gomega.BeNil())

		getRuntimesHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Success endpoint
			switch req.URL.Path {
			case expectedPathPrefix:
				switch req.URL.RawQuery {
				case "page=1":
					_, err := rw.Write(runtimesPage1Response)
					g.Expect(err).Should(gomega.BeNil())
					rw.WriteHeader(http.StatusOK)
					return

				case "page=2":
					_, err := rw.Write(runtimesPage2Response)
					g.Expect(err).Should(gomega.BeNil())
					rw.WriteHeader(http.StatusOK)
					return
				}
			}
		})

		// Start a local test HTTP server
		srv := kmctesting.StartTestServer(expectedPathPrefix, getRuntimesHandler, g)

		// Wait until test server is ready
		g.Eventually(func() int {
			// Ignoring error is ok as it goes for retry for non-200 cases
			healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
			g.Expect(err).Should(gomega.BeNil())
			return healthResp.StatusCode
		}, timeout).Should(gomega.Equal(http.StatusOK))

		kebURL := fmt.Sprintf("%s%s", srv.URL, expectedPathPrefix)
		kebClient := getKEBClient(kebURL)

		req, err := kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())

		// when
		gotRuntimes, err := kebClient.GetAllRuntimes(req)

		// then
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(*gotRuntimes).To(gomega.Equal(*expectedRuntimes))
		g.Expect(gotRuntimes.TotalCount).To(gomega.Equal(expectedRuntimes.TotalCount))
		g.Expect(len(gotRuntimes.Data)).To(gomega.Equal(4))

		// ensure metric exists.
		expectedNumberOfMetrics := 1 // because single request is send.
		expectedNumberOfLabels := 2  // because 2 labels are set in the definition of latencyMetric metric.
		g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))

		// check if the required labels exists in the metric.
		pMetric, err := kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
		gotLabel := pMetric.Metric[0].Label
		g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
		// response status label.
		statusLabel := kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
		g.Expect(statusLabel).ShouldNot(gomega.BeNil())
		g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusOK)))
		// request URL label.
		g.Expect(kmctesting.PrometheusFilterLabelPair(gotLabel, requestURLLabel)).ShouldNot(gomega.BeNil())

		// Testing http 404 for non-existent path
		// reset metrics state.
		latencyMetric.Reset()
		kebClient.Config.URL = fmt.Sprintf("%s/nopaging", kebClient.Config.URL)
		req, err = kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())
		_, err = kebClient.GetAllRuntimes(req)
		g.Expect(err).ShouldNot(gomega.BeNil())
		g.Expect(err.Error()).To(gomega.ContainSubstring("KEB returned status code: 404"))

		// ensure metric exists.
		g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))
		// check if the required labels exists in the metric.
		pMetric, err = kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
		gotLabel = pMetric.Metric[0].Label
		g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
		// response status label.
		statusLabel = kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
		g.Expect(statusLabel).ShouldNot(gomega.BeNil())
		g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusNotFound)))
	})

	t.Run("when all runtimes are returned in 1 page", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		// given
		latencyMetric.Reset()
		runtimesResponse, err := kmctesting.LoadFixtureFromFile(kebRuntimeResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		expectedRuntimes := new(runtime.RuntimesPage)
		err = json.Unmarshal(runtimesResponse, expectedRuntimes)
		g.Expect(err).Should(gomega.BeNil())

		getRuntimesHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Success endpoint
			g.Expect(req.URL.Path).To(gomega.Equal(expectedPathPrefixWith1Page))
			_, err := rw.Write(runtimesResponse)
			g.Expect(err).Should(gomega.BeNil())
			rw.WriteHeader(http.StatusOK)
		})

		// Start a local test HTTP server
		srv := kmctesting.StartTestServer(expectedPathPrefixWith1Page, getRuntimesHandler, g)

		// Wait until test server is ready
		g.Eventually(func() int {
			// Ignoring error is ok as it goes for retry for non-200 cases
			healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
			t.Logf("retrying :%v", err)
			return healthResp.StatusCode
		}, timeout).Should(gomega.Equal(http.StatusOK))

		kebURL := fmt.Sprintf("%s%s", srv.URL, expectedPathPrefixWith1Page)
		kebClient := getKEBClient(kebURL)

		// Testing response which contains all the records
		req, err := kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())

		// when
		gotRuntimes, err := kebClient.GetAllRuntimes(req)

		// then
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(*gotRuntimes).To(gomega.Equal(*expectedRuntimes))
		g.Expect(gotRuntimes.TotalCount).To(gomega.Equal(expectedRuntimes.TotalCount))
		g.Expect(len(gotRuntimes.Data)).To(gomega.Equal(4))

		// ensure metric exists.
		expectedNumberOfMetrics := 1 // because single request is send.
		expectedNumberOfLabels := 2  // because 2 labels are set in the definition of latencyMetric metric.
		g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))
		// check if the required labels exists in the metric.
		pMetric, err := kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
		gotLabel := pMetric.Metric[0].Label
		g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
		// response status label.
		statusLabel := kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
		g.Expect(statusLabel).ShouldNot(gomega.BeNil())
		g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusOK)))
	})

	t.Run("when HTTP non 2xx is returned by KEB", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)
		// given
		latencyMetric.Reset()
		getRuntimesHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Success endpoint
			g.Expect(req.URL.Path).To(gomega.Equal(expectedPathPrefixWith1Page))
			rw.WriteHeader(http.StatusInternalServerError)
		})

		// Start a local test HTTP server
		srv := kmctesting.StartTestServer(expectedPathPrefixWith1Page, getRuntimesHandler, g)

		// Wait until test server is ready
		g.Eventually(func() int {
			// Ignoring error is ok as it goes for retry for non-200 cases
			healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
			t.Logf("retrying :%v", err)
			return healthResp.StatusCode
		}, timeout).Should(gomega.Equal(http.StatusOK))

		kebURL := fmt.Sprintf("%s%s", srv.URL, expectedPathPrefixWith1Page)

		kebClient := getKEBClient(kebURL)

		// Testing response which contains HTTP 500
		req, err := kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())

		// when
		_, err = kebClient.GetAllRuntimes(req)

		// then
		g.Expect(err.Error()).Should(gomega.ContainSubstring("KEB returned status code: 500"))

		// ensure metric exists.
		expectedNumberOfMetrics := 1 // because single request is send.
		expectedNumberOfLabels := 2  // because 2 labels are set in the definition of latencyMetric metric.
		g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))
		// check if the required labels exists in the metric.
		pMetric, err := kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
		gotLabel := pMetric.Metric[0].Label
		g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
		// response status label.
		statusLabel := kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
		g.Expect(statusLabel).ShouldNot(gomega.BeNil())
		g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusInternalServerError)))
	})
}

func getKEBClient(url string) *Client {
	config := &Config{
		URL:              url,
		Timeout:          3 * time.Second,
		RetryCount:       1,
		PollWaitDuration: 10 * time.Minute,
	}
	return &Client{
		HTTPClient: http.DefaultClient,
		Logger:     logger.NewLogger(zapcore.InfoLevel),
		Config:     config,
	}
}
