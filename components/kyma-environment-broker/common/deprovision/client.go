package deprovision

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
)

const defaultPageSize = 100

// Client is the interface to interact with the KEB /deprovision API as an HTTP client using OIDC ID token in JWT format.
type Client interface {
	DeprovisionRuntime(params DeprovisionParameters) error
}

type DeprovisionClient struct {
	log    logrus.FieldLogger
	URL    string
	client *http.Client
}

func NewDeprovisionClient(parameters DeprovisionParameters) *DeprovisionClient {
	cfg := clientcredentials.Config{
		ClientID:     parameters.Oauth2ClientID,
		ClientSecret: parameters.Oauth2ClientSecret,
		TokenURL:     parameters.Oauth2IssuerURL,
		Scopes:       parameters.Scopes,
		AuthStyle:    parameters.AuthStyle,
	}
	httpClientOAuth := cfg.Client(parameters.Context)
	httpClientOAuth.Timeout = 30 * time.Second

	return &DeprovisionClient{
		log:    logrus.WithField("client", "deprovision"),
		URL:    parameters.EndpointURL,
		client: httpClientOAuth,
	}
}

func (c DeprovisionClient) DeprovisionRuntime(instanceID string) error {
	logrus.Info("DeprovisionRuntime is called")
	url := c.URL + "/oauth/v2/service_instances/" +
		instanceID + "?accepts_incomplete=true&service_id=faebbe18-0a84-11e5-ab14-d663bd873d97&plan_id=0c712d43-b1e6-470s-9fe5-8e1d552aa6a5"

	logrus.Infof("url: %s", url)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("while creating the HTTP Delete request for deprovisioning: %w", err)
	}
	request.Header.Set("X-Broker-API-Version", "2.14")

	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("while calling %s: %w", request.URL.String(), err)
	}

	cerr := response.Body.Close()
	if err == nil {
		err = cerr
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("calling %s returned %d (%s) status", request.URL.String(), response.StatusCode, response.Status)
	}
	logrus.Infof("Deprovisioning request returned code: " + response.Status)
	c.log.Infof("Deprovisioning request returned code: " + response.Status)

	return err
}
