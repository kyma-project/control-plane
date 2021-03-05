// +build sm_integration

package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"

	"fmt"
	"os"
	"testing"
	"time"

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
func TestClsProvisionSteps(t *testing.T) {
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

	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	fmt.Println("Start Testing")

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
	operation, retry, err := offeringStep.Run(operation, log)
	time.Sleep(10 * time.Second)
	fmt.Printf(">>> Offering %#v\n", operation.Cls)

	require.NoError(t, err)
	require.Zero(t, retry)

	clsClient := cls.NewClient(clsConfig)
	clsProvisioner := cls.NewProvisioner(db.CLSInstances(), clsClient)
	provisioningStep := NewClsProvisionStep(clsConfig, clsProvisioner, repo)
	operation, retry, err = provisioningStep.Run(operation, log)
	fmt.Printf(">>> Provisioning: %#v\n", operation.Cls)

	require.NoError(t, err)
	require.Zero(t, retry)

	bindingStep := NewClsBindStep(clsConfig, clsClient, repo, "1234567890123456")

	for i := 0; i < 200; i++ {
		time.Sleep(retry)
		operation, retry, err = bindingStep.Run(operation, log)
		fmt.Printf(">>> Binding: %#v\n", operation.Cls)

		require.NoError(t, err)
		if operation.Cls.Binding.Bound {
			break
		}
	}

	// Unbind
	deprovOperation := internal.DeprovisioningOperation{
		Operation: internal.Operation{ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: func(s string) *string { return &s }("westeurope"),
			},
		}},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
	}

	deprovOperation.Cls = operation.Cls
	repo.InsertDeprovisioningOperation(deprovOperation)

	unbindingStep := deprovisioning.NewClsUnbindStep(clsConfig, repo)
	deprovOp, retry, err := unbindingStep.Run(deprovOperation, log)
	fmt.Printf(">>> UnBinding: %#v\n", deprovOp.Cls)

	clsDeprovisioner := cls.NewDeprovisioner(db.CLSInstances(), clsClient, log)
	deprovisioningStep := deprovisioning.NewClsDeprovisionStep(clsConfig, repo, clsDeprovisioner)

	for i := 0; i < 10; i++ {
		op, offset, err := deprovisioningStep.Run(deprovOperation, log)
		require.NoError(t, err)
		deprovOperation = op

		if !deprovOperation.Cls.Instance.Provisioned {
			require.Empty(t, deprovOperation.Cls.Instance.InstanceID)
			break
		}

		time.Sleep(offset)
	}

	fmt.Printf(">>> Deprovisioning: %#v\n", operation.Cls)

	require.NoError(t, err)
}
