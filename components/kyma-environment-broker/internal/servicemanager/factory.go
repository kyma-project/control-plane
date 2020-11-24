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
func (f *ClientFactory) ForCustomerCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (Client, error) {
	credentials, err := f.ProvideCredentials(reqCredentials, log)
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
func (f *ClientFactory) ForCredentials(creds Credentials) Client {
	return NewWithHttpClient(creds, f.httpClient)
}

func (f *ClientFactory) ProvideCredentials(reqCredentials *Credentials, log logrus.FieldLogger) (*Credentials, error) {
	if f.shouldOverride(reqCredentials) {
		log.Infof("Overrides ServiceManager credentials")
		return &Credentials{
			Username: f.config.Username,
			Password: f.config.Password,
			URL:      f.config.URL,
		}, nil
	}
	if reqCredentials == nil {
		log.Warnf("Service Manager Credentials are required to be send in provisioning request (override_mode: %q)", f.config.OverrideMode)
		return nil, errors.New("Service Manager Credentials are required to be send in provisioning request.")
	}
	log.Infof("Provides customer ServiceManager credentials")
	return reqCredentials, nil
}

func (f *ClientFactory) shouldOverride(reqCredentials *Credentials) bool {
	if f.config.OverrideMode == SMOverrideModeAlways {
		return true
	}

	if f.config.OverrideMode == SMOverrideModeWhenNotSentInRequest && reqCredentials == nil {
		return true
	}

	return false
}
