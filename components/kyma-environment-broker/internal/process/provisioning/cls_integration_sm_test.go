// +build sm_integration

package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

// TestClsStepsWithRealServiceManager tests all CLS steps with real Service Manager
// Usage:
// export SM_URL=
// export SM_USERNAME=
// export SM_PASSWORD=
// export SAML_EXCHANGE_KEY=
// export SAML_SIGNATURE_PRIVATE_KEY=
// go test -v -tags=sm_integration -run TestClsStepsWithRealServiceManager -timeout 30m
func TestClsStepsWithRealServiceManager(t *testing.T) {
	clsConfig := &cls.Config{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		ServiceManager: &cls.ServiceManagerConfig{
			Credentials: []*cls.ServiceManagerCredentials{
				{
					Region:   "eu",
					URL:      os.Getenv("SM_URL"),
					Username: os.Getenv("SM_USERNAME"),
					Password: os.Getenv("SM_PASSWORD"),
				},
			},
		},
		SAML: &cls.SAMLConfig{
			AdminGroup:  "runtimeAdmin",
			ExchangeKey: os.Getenv("SAML_EXCHANGE_KEY"),
			RolesKey:    "groups",
			Idp: &cls.SAMLIdpConfig{
				EntityID:    "https://kymatest.accounts400.ondemand.com",
				MetadataURL: "https://kymatest.accounts400.ondemand.com/saml2/metadata",
			},
			Sp: &cls.SAMLSpConfig{
				EntityID:            "cls-dev",
				SignaturePrivateKey: os.Getenv("SAML_SIGNATURE_PRIVATE_KEY"),
			},
		},
	}

	db := storage.NewMemoryStorage()
	smClientFactory := servicemanager.NewClientFactory(servicemanager.Config{})

	runClsEndToEndFlow(t, clsConfig, db, smClientFactory)
}
