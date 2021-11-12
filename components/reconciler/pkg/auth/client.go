package auth

import (
	"net/http"

	mothership "github.com/kyma-project/control-plane/components/reconciler/pkg"
)

func NewClient(url string, httpClient *http.Client) (*mothership.Client, error) {
	client, err := mothership.NewClient(url)
	mothership.ParsePostOperationsSchedulingIDCorrelationIDStopResponse()
	if err != nil {
		return nil, err
	}
	client.Client = httpClient

	return client, nil
}
