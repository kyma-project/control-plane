package keb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
)

type Client struct {
	HTTPClient *http.Client
	Logger     *zap.SugaredLogger
	Config     *Config
}

const (
	backOffJitter = 0.1
	backOffFactor = 5.0

	clientName = "keb-client"
)

func NewClient(config *Config, logger *zap.SugaredLogger) *Client {
	kebHTTPClient := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   config.Timeout,
	}
	return &Client{
		HTTPClient: kebHTTPClient,
		Logger:     logger,
		Config:     config,
	}
}

func (c Client) NewRequest() (*http.Request, error) {
	kebURL, err := url.ParseRequestURI(c.Config.URL)
	if err != nil {
		return nil, err
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    kebURL,
	}
	return req, nil
}

func (c Client) GetAllRuntimes(req *http.Request) (*kebruntime.RuntimesPage, error) {
	morePages := true
	pageNum := 1
	recordsSeen := 0
	finalRuntimesPage := new(kebruntime.RuntimesPage)
	for morePages {
		runtimesPage, err := c.getRuntimesPerPage(req, pageNum)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get runtimes from KEB")
		}
		finalRuntimesPage.Data = append(finalRuntimesPage.Data, runtimesPage.Data...)
		finalRuntimesPage.Count = len(finalRuntimesPage.Data)
		recordsSeen += runtimesPage.Count
		c.namedLogger().Debugf("count: %d, records-seen: %d, page-num: %d, total-count: %d", runtimesPage.Count, recordsSeen, pageNum, runtimesPage.TotalCount)
		if recordsSeen >= runtimesPage.TotalCount {
			morePages = false
			continue
		}
		pageNum += 1
	}
	finalRuntimesPage.TotalCount = recordsSeen
	return finalRuntimesPage, nil
}

func (c Client) getRuntimesPerPage(req *http.Request, pageNum int) (*kebruntime.RuntimesPage, error) {
	c.Logger.Debugf("polling for runtimes with URL: %s", req.URL.String())
	query := url.Values{
		"page": []string{fmt.Sprintf("%d", pageNum)},
	}
	req.URL.RawQuery = query.Encode()
	customBackoff := wait.Backoff{
		Steps:    c.Config.RetryCount,
		Duration: c.HTTPClient.Timeout,
		Factor:   backOffFactor,
		Jitter:   backOffJitter,
	}
	var resp *http.Response
	var err error
	err = retry.OnError(customBackoff, func(err error) bool {
		return err != nil
	}, func() (err error) {
		metricTimer := prometheus.NewTimer(sentRequestDuration)
		resp, err = c.HTTPClient.Do(req)
		metricTimer.ObserveDuration()
		if err != nil {
			c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
				With(log.KeyRetry, log.ValueTrue).Warn("getting runtimes from KEB")
		}
		return
	})
	if resp != nil {
		totalRequest.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode)).Inc()
	}

	if err != nil {
		c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Warnw("getting runtimes from KEB")
		return nil, errors.Wrapf(err, "failed to get runtimes from KEB")
	}

	if resp.StatusCode != http.StatusOK {
		failedErr := fmt.Errorf("KEB returned status code: %d", resp.StatusCode)
		c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, failedErr.Error()).Error("get runtimes from KEB")
		return nil, failedErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Error("read response body")
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			if err = resp.Body.Close(); err != nil {
				c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
					Error("close body for KEB runtime request")
			}
		}
	}()
	runtimesPage := new(kebruntime.RuntimesPage)
	if err := json.Unmarshal(body, runtimesPage); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal runtimes response")
	}

	return runtimesPage, nil
}

func (c *Client) namedLogger() *zap.SugaredLogger {
	return c.Logger.Named(clientName).With("component", "KEB")
}
