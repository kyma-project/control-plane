package edp

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

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
)

func TestClient(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	dataTenant := "testTenant"
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

	edpClient := NewClient(config, logrus.New())
	testData := []byte("foodata")
	gotReq, err := edpClient.NewRequest(dataTenant)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(gotReq.URL.Host).To(gomega.Equal(edpURL.Host))

	resp, err := edpClient.Send(gotReq, testData)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(resp.StatusCode).Should(gomega.Equal(http.StatusCreated))

}

func TestClientRetry(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	dataTenant := "testTenant"
	expectedPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events", testNamespace, testDataStreamName, testDataStreamVersion, testTenant, testEnv)

	countRetry := 0
	counter := &countRetry
	expectedCountRetry := 2
	edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		g.Expect(req.URL.Path).To(gomega.Equal(expectedPath))
		g.Expect(req.Method).To(gomega.Equal(http.MethodPost))
		*counter += 1
		rw.WriteHeader(http.StatusInternalServerError)
	})

	srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, g)
	// Close the server when test finishes
	defer srv.Close()
	config := NewTestConfig(srv.URL)

	edpURL, err := url.ParseRequestURI(srv.URL)
	g.Expect(err).Should(gomega.BeNil())

	edpClient := NewClient(config, logrus.New())
	testData := []byte("foodata")
	gotReq, err := edpClient.NewRequest(dataTenant)
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(gotReq.URL.Host).To(gomega.Equal(edpURL.Host))

	_, err = edpClient.Send(gotReq, testData)
	g.Expect(err).ShouldNot(gomega.BeNil())
	g.Expect(err.Error()).Should(gomega.Equal("failed to POST event to EDP: failed to send event stream as EDP returned HTTP: 500"))
	g.Expect(countRetry).Should(gomega.Equal(expectedCountRetry))
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
