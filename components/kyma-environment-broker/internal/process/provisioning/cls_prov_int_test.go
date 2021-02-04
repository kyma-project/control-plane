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
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestClsSteps -count=1
func TestClsSteps(t *testing.T) {
	//db := storage.NewMemoryStorage()
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})
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

	offeringStep := NewClsOfferingStep(clsConfig, repo)
	//clsClient := cls.NewClient(logs.WithField("service", "clsClient"))
	//clsIM := cls.NewInstanceManager(db.CLSInstances(), clsClient, logs.WithField("service", "clsInstanceManager"))
	//instanceStep := NewProvideClsInstaceStep(clsIM, repo, "region", false)

	pp := internal.ProvisioningParameters{
		Parameters: internal.ProvisioningParametersDTO{
			Region: func() *string {
				local := "westeurope"
				return &local
			}(),
		},
	}
	operation := internal.ProvisioningOperation{
		Operation:       internal.Operation{ProvisioningParameters: pp},
		SMClientFactory: cliFactory,
	}

	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator

	repo.InsertProvisioningOperation(operation)

	log := logrus.New()

	operation, retry, err := offeringStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Cls)
	require.NoError(t, err)
	require.Zero(t, retry)

	// operation, retry, err = instanceStep.Run(operation, log)
	// fmt.Printf(">>> %#v\n", operation.Cls)
	// require.NoError(t, err)
	// require.Zero(t, retry)
}
