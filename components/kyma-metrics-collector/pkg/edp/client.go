package edp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sirupsen/logrus"
)

type Client struct {
	HttpClient *http.Client
	Config     *Config
	Logger     *logrus.Logger
}

const (
	edpPathFormat          = "%s/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events"
	contentType            = "application/json;charset=utf-8"
	userAgentKMC           = "kyma-metrics-collector"
	userAgentKeyHeader     = "User-Agent"
	contentTypeKeyHeader   = "Content-Type"
	authorizationKeyHeader = "Authorization"
)

func NewClient(config *Config, logger *logrus.Logger) *Client {
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
	edpURL := fmt.Sprintf(edpPathFormat,
		eClient.Config.URL,
		eClient.Config.Namespace,
		eClient.Config.DataStreamName,
		eClient.Config.DataStreamVersion,
		dataTenant,
		eClient.Config.DataStreamEnv,
	)

	req, err := http.NewRequest(http.MethodPost, edpURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed generate request for EDP, %d: %v", http.StatusBadRequest, err)
	}

	req.Header.Set(userAgentKeyHeader, userAgentKMC)
	req.Header.Add(contentTypeKeyHeader, contentType)
	req.Header.Add(authorizationKeyHeader, fmt.Sprintf("Bearer %s", eClient.Config.Token))

	return req, nil
}

func (eClient Client) Send(req *http.Request, payload []byte) (*http.Response, error) {
	metricTimer := prometheus.NewTimer(sentRequestDuration)

	var resp *http.Response
	var err error
	customBackoff := wait.Backoff{
		Steps:    eClient.Config.EventRetry,
		Duration: eClient.Config.Timeout,
		Factor:   5.0,
		Jitter:   0.1,
	}
	err = retry.OnError(customBackoff, func(err error) bool {
		if err != nil {
			return true
		}
		return false
	}, func() (err error) {
		req.Body = ioutil.NopCloser(bytes.NewReader(payload))
		resp, err = eClient.HttpClient.Do(req)
		if err != nil {
			eClient.Logger.Debugf("req: %v", req)
			eClient.Logger.Warnf("will be retried: failed to send event stream to EDP: %v", err)
			return
		}

		if resp.StatusCode != http.StatusCreated {
			non2xxErr := fmt.Errorf("failed to send event stream as EDP returned HTTP: %d", resp.StatusCode)
			eClient.Logger.Warnf("will be retried: %v", non2xxErr)
			err = non2xxErr
		}
		return
	})
	metricTimer.ObserveDuration()
	totalRequest.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode)).Inc()

	if err != nil {
		return nil, errors.Wrapf(err, "failed to POST event to EDP")
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			eClient.Logger.Warn(err)
		}
	}()

	eClient.Logger.Debugf("sent an event to '%s' with eventstream: '%s'", req.URL.String(), string(payload))
	return resp, nil
}
