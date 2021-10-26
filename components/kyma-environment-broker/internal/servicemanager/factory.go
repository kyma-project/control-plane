package servicemanager

import (
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	errors "github.com/pkg/errors"
)

type ClientFactory struct {
	config Config

	httpClient *http.Client
}

type Credentials struct {
	Username string
	Password string
	URL      string
}

type RequestContext struct {
	SubaccountID string
	Credentials  *Credentials
}

func NewClientFactory(cfg Config) *ClientFactory {
	return &ClientFactory{
		config: cfg,
		httpClient: &http.Client{
			Transport: nil,
			Timeout:   30 * time.Second,
		},
	}
}

func (c Credentials) WithNormalizedURL() Credentials {
	url := strings.TrimSuffix(c.URL, "/")
	return Credentials{
		Username: c.Username,
		Password: c.Password,
		URL:      url,
	}
}

// ForCustomerCredentials provides a client with request Credentials (see internal.ProvisioningParameters.ErsContext).
// Those Credentials could be overridden based on KEB configuration (OverrideMode).
func (f *ClientFactory) ForCustomerCredentials(request RequestContext, log logrus.FieldLogger) (Client, error) {
	credentials, err := f.ProvideCredentials(request, log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create Service Manager client")
	}
	return &client{
		creds:      *credentials,
		httpClient: f.httpClient,
	}, nil
}

// put here methods which creates SM client for different Credentials
// ...
func (f *ClientFactory) ForCredentials(credentials *Credentials) Client {
	return &client{
		creds:      *credentials,
		httpClient: f.httpClient,
	}
}

func (f *ClientFactory) ProvideCredentials(request RequestContext, log logrus.FieldLogger) (*Credentials, error) {
	if f.shouldOverride(request) {
		log.Infof("Overrides ServiceManager credentials")
		return &Credentials{
			Username: f.config.Username,
			Password: f.config.Password,
			URL:      f.config.URL,
		}, nil
	}
	if request.Credentials == nil {
		log.Warnf("Service Manager Credentials are required to be send in provisioning request (override_mode: %q)", f.config.OverrideMode)
		return nil, errors.New("Service Manager Credentials are required to be send in provisioning request.")
	}
	log.Infof("Provides customer ServiceManager credentials")
	return request.Credentials, nil
}

func (f *ClientFactory) shouldOverride(request RequestContext) bool {
	if f.config.SubaccountWithRequestCredentials != "" && request.SubaccountID == f.config.SubaccountWithRequestCredentials {
		return false
	}

	if f.config.OverrideMode == SMOverrideModeAlways {
		return true
	}

	if f.config.OverrideMode == SMOverrideModeWhenNotSentInRequest && request.Credentials == nil {
		return true
	}

	return false
}
