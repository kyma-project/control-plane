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
	}

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: func(s string) *string { return &s }("westeurope"),
			},
		}},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
		InputCreator:    newInputCreator(),
	}

	repo := storage.NewMemoryStorage().Operations()
	repo.InsertProvisioningOperation(operation)

	offeringStep := NewClsOfferingStep(clsConfig, repo)
	clsClient := cls.NewClient(logs.WithField("service", "clsClient"))
	clsIM := cls.NewInstanceManager(db.CLSInstances(), clsClient, logs.WithField("service", "clsInstanceManager"))
	provisioningStep := NewProvideClsInstaceStep(clsIM, repo, "region", false)

	operation, retry, err := offeringStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Cls)
	require.NoError(t, err)
	require.Zero(t, retry)

	// operation, retry, err = provisioningStep.Run(operation, log)
	// fmt.Printf(">>> %#v\n", operation.Cls)
	// require.NoError(t, err)
	// require.Zero(t, retry)
}
