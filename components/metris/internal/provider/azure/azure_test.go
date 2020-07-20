package azure

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/stretchr/testify/assert"
)

var (
	noopLogger     = log.NewNoopLogger()
	providerConfig = &provider.Config{
		PollInterval:     time.Minute,
		Workers:          1,
		Buffer:           1,
		ClientTraceLevel: 2,
		ClusterChannel:   make(chan *gardener.Cluster, 1),
		EventsChannel:    make(chan *edp.Event, 1),
		Logger:           noopLogger,
	}

	testCluster = &gardener.Cluster{
		TechnicalID:  "test-technicalid",
		ProviderType: "az",
		CredentialData: map[string][]byte{
			"clientID":       []byte("test-clientid"),
			"clientSecret":   []byte("test-clientsecret"),
			"subscriptionID": []byte("test-subscriptionid"),
			"tenantID":       []byte("test-tenantid"),
		},
		AccountID:    "test-accountid",
		SubAccountID: "test-subaccountid",
	}
)

func TestNewAzureProvider(t *testing.T) {
	p := NewAzureProvider(providerConfig)
	assert.Implements(t, (*provider.Provider)(nil), p, "")
}
