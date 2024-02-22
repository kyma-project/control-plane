package edp

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap/zapcore"

	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
)

const (
	timeout               = 5 * time.Second
	testTenant            = "testTenant"
	testDataStreamName    = "dataStream"
	testNamespace         = "namespace"
	testDataStreamVersion = "v1"
	testToken             = "token"
	testEnv               = "env"

	//Metrics related variable
	histogramName = "kmc_edp_request_duration_seconds"
)

func TestClient(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// given
	dataTenant := "testTenant"
	// reset metrics state.
	latencyMetric.Reset()

	expectedPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events", testNamespace, testDataStreamName, testDataStreamVersion, testTenant, testEnv)
	expectedHeaders := http.Header{
		"Authorization":   []string{fmt.Sprintf("Bearer %s", testToken)},
		"Accept-Encoding": []string{"gzip"},
		"User-Agent":      []string{"kyma-metrics-collector"},
		"Content-Type":    []string{"application/json;charset=utf-8"},
	}
	edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		g.Expect(req.Header).To(gomega.Equal(expectedHeaders))
		g.Expect(req.URL.Path).To(gomega.Equal(expectedPath))
		g.Expect(req.Method).To(gomega.Equal(http.MethodPost))
		rw.WriteHeader(http.StatusCreated)
	})

	srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, g)
	// Close the server when test finishes
	defer srv.Close()
	config := NewTestConfig(srv.URL)

	edpURL, err := url.ParseRequestURI(srv.URL)
	g.Expect(err).Should(gomega.BeNil())

	edpClient := NewClient(config, logger.NewLogger(zapcore.InfoLevel))
	testData := []byte("foodata")
	gotReq, err := edpClient.NewRequest(dataTenant)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(gotReq.URL.Host).To(gomega.Equal(edpURL.Host))

	// when
	resp, err := edpClient.Send(gotReq, testData)

	// then
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(resp.StatusCode).Should(gomega.Equal(http.StatusCreated))

	// ensure metrics exists.
	g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))

	// check if the required labels exists in the metric.
	expectedNumberOfMetrics := 1 // because single request is send.
	expectedNumberOfLabels := 2  // because 2 labels are set in the definition of latencyMetric metric.
	pMetric, err := kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
	gotLabel := pMetric.Metric[0].Label
	g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
	// response status label.
	statusLabel := kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
	g.Expect(statusLabel).ShouldNot(gomega.BeNil())
	g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusCreated)))
	// request URL label.
	g.Expect(kmctesting.PrometheusFilterLabelPair(gotLabel, requestURLLabel)).ShouldNot(gomega.BeNil())
}

func TestClientRetry(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// given
	dataTenant := "testTenant"
	expectedPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events", testNamespace, testDataStreamName, testDataStreamVersion, testTenant, testEnv)
	// reset metrics state.
	latencyMetric.Reset()

	counter := 0
	expectedCountRetry := 2
	edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		g.Expect(req.URL.Path).To(gomega.Equal(expectedPath))
		g.Expect(req.Method).To(gomega.Equal(http.MethodPost))
		counter += 1
		rw.WriteHeader(http.StatusInternalServerError)
	})
	srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, g)
	// Close the server when test finishes
	defer srv.Close()
	config := NewTestConfig(srv.URL)

	edpURL, err := url.ParseRequestURI(srv.URL)
	g.Expect(err).Should(gomega.BeNil())

	edpClient := NewClient(config, logger.NewLogger(zapcore.InfoLevel))
	testData := []byte("foodata")
	gotReq, err := edpClient.NewRequest(dataTenant)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(gotReq.URL.Host).To(gomega.Equal(edpURL.Host))

	// when
	_, err = edpClient.Send(gotReq, testData)

	// then
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(err.Error()).Should(gomega.ContainSubstring("failed to send event stream as EDP returned HTTP: 500"))
	g.Expect(counter).Should(gomega.Equal(expectedCountRetry))

	// ensure metric exists.
	expectedNumberOfMetrics := 1 // because single request is send.
	expectedNumberOfLabels := 2  // because 2 labels are set in the definition of latencyMetric metric.
	g.Expect(testutil.CollectAndCount(latencyMetric, histogramName)).Should(gomega.Equal(1))

	// ensure metric has expected label value.
	pMetric, err := kmctesting.PrometheusGatherAndReturn(latencyMetric, histogramName)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(pMetric.Metric).Should(gomega.HaveLen(expectedNumberOfMetrics))
	gotLabel := pMetric.Metric[0].Label
	g.Expect(gotLabel).Should(gomega.HaveLen(expectedNumberOfLabels))
	statusLabel := kmctesting.PrometheusFilterLabelPair(gotLabel, responseCodeLabel)
	g.Expect(statusLabel).ShouldNot(gomega.BeNil())
	g.Expect(statusLabel.GetValue()).Should(gomega.Equal(fmt.Sprint(http.StatusInternalServerError)))
}

func NewTestConfig(url string) *Config {
	return &Config{
		URL:               url,
		Token:             testToken,
		Namespace:         testNamespace,
		DataStreamName:    testDataStreamName,
		DataStreamVersion: testDataStreamVersion,
		DataStreamEnv:     testEnv,
		Timeout:           timeout,
		EventRetry:        2,
	}
}
