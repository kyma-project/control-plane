package keb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	kmctesting "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
)

const (
	timeout                         = 5 * time.Second
	expectedPathPrefix              = "/runtimes"
	expectedPathPrefixWith1Page     = "/runtimes/with1page"
	kebRuntimeResponseFilePath      = "../testing/fixtures/runtimes_response.json"
	kebRuntimePage1ResponseFilePath = "../testing/fixtures/runtimes_response_page1.json"
	kebRuntimePage2ResponseFilePath = "../testing/fixtures/runtimes_response_page2.json"
)

func TestGetAllRuntimes(t *testing.T) {

	t.Run("when 2 pages are returned for all runtimes on matching path and HTTP 404 for non matched ones", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)

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

		config := &Config{
			URL:              kebURL,
			Timeout:          3 * time.Second,
			RetryCount:       1,
			PollWaitDuration: 10 * time.Minute,
		}
		kebClient := &Client{
			HTTPClient: http.DefaultClient,
			Logger:     &logrus.Logger{},
			Config:     config,
		}

		req, err := kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())

		gotRuntimes, err := kebClient.GetAllRuntimes(req)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(*gotRuntimes).To(gomega.Equal(*expectedRuntimes))
		g.Expect(gotRuntimes.TotalCount).To(gomega.Equal(expectedRuntimes.TotalCount))
		g.Expect(len(gotRuntimes.Data)).To(gomega.Equal(4))

		// Testing http 404 for non existent path
		config.URL = fmt.Sprintf("%s/nopaging", kebClient.Config.URL)
		req, err = kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())
		_, err = kebClient.GetAllRuntimes(req)
		g.Expect(err).ShouldNot(gomega.BeNil())
		g.Expect(err.Error()).To(gomega.Equal("failed to get runtimes from KEB: KEB returned status code: 404"))
	})

	t.Run("when all runtimes are returned in 1 page", func(t *testing.T) {
		g := gomega.NewGomegaWithT(t)

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

		config := &Config{
			URL:              kebURL,
			Timeout:          3 * time.Second,
			RetryCount:       1,
			PollWaitDuration: 10 * time.Minute,
		}
		kebClient := &Client{
			HTTPClient: http.DefaultClient,
			Logger:     &logrus.Logger{},
			Config:     config,
		}

		req, err := kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())

		// Testing response which contains all the records
		req, err = kebClient.NewRequest()
		g.Expect(err).Should(gomega.BeNil())
		gotRuntimes, err := kebClient.GetAllRuntimes(req)
		g.Expect(err).Should(gomega.BeNil())
		g.Expect(*gotRuntimes).To(gomega.Equal(*expectedRuntimes))
		g.Expect(gotRuntimes.TotalCount).To(gomega.Equal(expectedRuntimes.TotalCount))
		g.Expect(len(gotRuntimes.Data)).To(gomega.Equal(4))
	})
}
