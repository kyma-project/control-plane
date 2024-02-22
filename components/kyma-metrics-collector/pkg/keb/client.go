package keb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/avast/retry-go/v4"
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
)

type Client struct {
	HTTPClient *http.Client
	Logger     *zap.SugaredLogger
	Config     *Config
}

const (
	clientName    = "keb-client"
	retryInterval = 10 * time.Second
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
		c.namedLogger().Debugf("count: %d, records-seen: %d, page-num: %d, total-count: %d",
			runtimesPage.Count, recordsSeen, pageNum, runtimesPage.TotalCount)
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
	// define URL.
	c.Logger.Debugf("polling for runtimes with URL: %s", req.URL.String())
	query := url.Values{
		"page": []string{fmt.Sprintf("%d", pageNum)},
	}
	req.URL.RawQuery = query.Encode()

	// define request retry options.
	retryOptions := []retry.Option{
		retry.Attempts(uint(c.Config.RetryCount)),
		retry.Delay(retryInterval),
	}

	// send request with retries.
	runtimesPage, err := retry.DoWithData(
		func() (*kebruntime.RuntimesPage, error) {
			// send request and record duration.
			reqStartTime := time.Now()
			resp, err := c.HTTPClient.Do(req)
			duration := time.Since(reqStartTime)

			// check for error.
			if err != nil {
				c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
					With(log.KeyRetry, log.ValueTrue).Warn("getting runtimes from KEB")
				// record metric.
				responseCode := http.StatusBadRequest
				urlErr := err.(*url.Error)
				if urlErr.Timeout() {
					responseCode = http.StatusRequestTimeout
				}
				recordKEBLatency(duration, responseCode, c.Config.URL)
				// return error.
				return nil, err
			}

			// record metric.
			recordKEBLatency(duration, resp.StatusCode, c.Config.URL)

			// defer to close response body.
			defer func() {
				if errClose := resp.Body.Close(); errClose != nil {
					c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, errClose.Error()).
						Error("close body for KEB runtime request")
				}
			}()

			// return error object if status is not OK.
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("KEB returned status code: %d", resp.StatusCode)
			}

			// read data from response body.
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError,
					err.Error()).Error("read response body")
				return nil, err
			}

			// parse body as RuntimesPage object.
			runtimesPage := new(kebruntime.RuntimesPage)
			if err = json.Unmarshal(body, runtimesPage); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal runtimes response")
			}
			return runtimesPage, err
		},
		retryOptions...,
	)
	return runtimesPage, err
}

func (c *Client) namedLogger() *zap.SugaredLogger {
	return c.Logger.Named(clientName).With("component", "KEB")
}
