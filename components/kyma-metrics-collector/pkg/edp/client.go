package edp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
)

type Client struct {
	HttpClient *http.Client
	Config     *Config
	Logger     *zap.SugaredLogger
}

const (
	edpPathFormat          = "%s/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events"
	contentType            = "application/json;charset=utf-8"
	userAgentKMC           = "kyma-metrics-collector"
	userAgentKeyHeader     = "User-Agent"
	contentTypeKeyHeader   = "Content-Type"
	authorizationKeyHeader = "Authorization"
	clientName             = "edp-client"
	tenantIdPlaceholder    = "<subAccountId>"
)

func NewClient(config *Config, logger *zap.SugaredLogger) *Client {
	httpClient := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   config.Timeout,
	}
	return &Client{
		HttpClient: httpClient,
		Logger:     logger,
		Config:     config,
	}
}

func (eClient Client) NewRequest(dataTenant string) (*http.Request, error) {
	edpURL := eClient.getEDPURL(dataTenant)

	req, err := http.NewRequest(http.MethodPost, edpURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed generate request for EDP, %d: %v", http.StatusBadRequest, err)
	}

	req.Header.Set(userAgentKeyHeader, userAgentKMC)
	req.Header.Add(contentTypeKeyHeader, contentType)
	req.Header.Add(authorizationKeyHeader, fmt.Sprintf("Bearer %s", eClient.Config.Token))

	return req, nil
}

func (eClient Client) getEDPURL(dataTenant string) string {
	return fmt.Sprintf(edpPathFormat,
		eClient.Config.URL,
		eClient.Config.Namespace,
		eClient.Config.DataStreamName,
		eClient.Config.DataStreamVersion,
		dataTenant,
		eClient.Config.DataStreamEnv,
	)
}

func (eClient Client) Send(req *http.Request, payload []byte) (*http.Response, error) {
	var resp *http.Response
	var err error
	customBackoff := wait.Backoff{
		Steps:    eClient.Config.EventRetry,
		Duration: eClient.Config.Timeout,
		Factor:   5.0,
		Jitter:   0.1,
	}
	err = retry.OnError(customBackoff, func(err error) bool {
		return err != nil
	}, func() (err error) {
		reqStartTime := time.Now()
		// send request.
		req.Body = io.NopCloser(bytes.NewReader(payload))
		resp, err = eClient.HttpClient.Do(req)
		duration := time.Since(reqStartTime)
		// check result.
		if err != nil {
			urlErr := err.(*url.Error)
			responseCode := http.StatusBadRequest
			if urlErr.Timeout() {
				responseCode = http.StatusRequestTimeout
			}
			// record metric.
			recordEDPLatency(duration, responseCode, eClient.getEDPURL(tenantIdPlaceholder))
			// log error.
			eClient.namedLogger().Debugf("req: %v", req)
			eClient.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
				With(log.KeyRetry, log.ValueTrue).Warn("send event stream to EDP")
			return
		}

		if resp.StatusCode != http.StatusCreated {
			non2xxErr := fmt.Errorf("failed to send event stream as EDP returned HTTP: %d", resp.StatusCode)
			eClient.namedLogger().With(log.KeyError, non2xxErr.Error()).With(log.KeyRetry, log.ValueTrue).
				Warn("send event stream as EDP")
			err = non2xxErr
		}

		// record metric.
		// the request URL is recorded without the actual tenant id to avoid having multiple histograms.
		recordEDPLatency(duration, resp.StatusCode, eClient.getEDPURL(tenantIdPlaceholder))
		return
	})

	if err != nil {
		return nil, errors.Wrapf(err, "failed to POST event to EDP")
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			eClient.namedLogger().Warn(err)
		}
	}()

	eClient.namedLogger().Debugf("sent an event to '%s' with eventstream: '%s'", req.URL.String(), string(payload))
	return resp, nil
}

func (c *Client) namedLogger() *zap.SugaredLogger {
	return c.Logger.Named(clientName).With("component", "EDP")
}
