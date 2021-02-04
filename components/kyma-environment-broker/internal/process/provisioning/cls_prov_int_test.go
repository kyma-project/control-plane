// +build sm_integration

package provisioning

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestClsSteps tests all CLS steps with real Service Manager
// Usage:
// export SM_URL=
// export SM_USERNAME=
// export SM_PASSWORD=
// export SAML_EXCHANGE_KEY=
// export SAML_SIGNATURE_PRIVATE_KEY
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestClsSteps -count=1
func TestClsSteps(t *testing.T) {
	clsConfig := &cls.Config{
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

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: func(s string) *string { return &s }("westeurope"),
			},
		}},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
		InputCreator:    newInputCreator(),
	}

	db := storage.NewMemoryStorage()
	repo := db.Operations()
	repo.InsertProvisioningOperation(operation)

	offeringStep := NewClsOfferingStep(clsConfig, repo)

	creator := cls.NewClient(clsConfig, logger)
	instanceManager := cls.NewInstanceManager(db.CLSInstances(), creator, logger)
	provisioningStep := NewClsProvisioningStep(clsConfig, instanceManager, repo)

	operation, retry, err := offeringStep.Run(operation, logger)
	fmt.Printf(">>> %#v\n", operation.Cls)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = provisioningStep.Run(operation, logger)
	fmt.Printf(">>> %#v\n", operation.Cls)
	require.NoError(t, err)
	require.Zero(t, retry)
}
