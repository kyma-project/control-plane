// +build sm_integration

package deprovisioning

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestProvisioningSteps tests all Ems steps with real Service Manager
// Usage:
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// export INSTANCE_ID=
// export BINDING_ID=
// export BROKER_ID=
// export SERVICE_ID=
// export PLAN_ID=
// go test -v -tags=sm_integration ./internal/process/deprovisioning/... -run TestDeprovisioningSteps -count=1
func TestClsDeprovisionSteps(t *testing.T) {
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

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{ProvisioningParameters: internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: func(s string) *string { return &s }("westeurope"),
			},
		}},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
	}
	operation.Cls = internal.ClsData{
		Instance: internal.ServiceManagerInstanceInfo{
			BrokerID:              "d9378dcd-8a21-4986-9f1d-4a3004e4cfd0", // saved in InstanceKey, see the provisioning step
			ServiceID:             "8ab39896-b78a-11ea-b3de-0242ac130004",
			PlanID:                "b0d143b6-23b4-4215-b5a6-17b69a9e989a",
			InstanceID:            "c7739986-a493-43dc-9a38-684f215be7b0",
			Provisioned:           true,
			ProvisioningTriggered: false,
		},
		BindingID: os.Getenv("BINDING_ID"),
		Overrides: "encryptedEventingOverrides",
	}

	db := storage.NewMemoryStorage()
	repo := db.Operations()
	repo.InsertDeprovisioningOperation(operation)

	log := logrus.New()

	deprovisioningStep := NewClsDeprovisionStep(clsConfig, repo)

	operation, retry, err := deprovisioningStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Cls)
	require.NoError(t, err)
	require.Zero(t, retry)
}
