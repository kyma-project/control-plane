// +build sm_integration

package deprovisioning

import (
	"os"
	"testing"
	"time"

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
// export BROKER_ID=
// export SERVICE_ID=
// export PLAN_ID=
// export INSTANCE_ID=
// go test -v -tags=sm_integration ./internal/process/deprovisioning/... -run TestClsDeprovisionSteps -count=1
func TestClsDeprovisionSteps(t *testing.T) {
	var (
		globalAccountID = "fake-global-account-id"
		skrInstanceID   = "fake-skr-instance-id"
	)

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

	instance := internal.NewCLSInstance(globalAccountID, "eu", internal.WithID(os.Getenv("INSTANCE_ID")), internal.WithReferences(skrInstanceID))
	db := storage.NewMemoryStorage()
	clsStorage := db.CLSInstances()
	clsStorage.Insert(*instance)

	operationStorage := db.Operations()
	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceID: skrInstanceID,
			ProvisioningParameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{GlobalAccountID: globalAccountID}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Region:    "eu",
					Overrides: "bsae64-encrypted-overrides",
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:              os.Getenv("BROKER_ID"), // saved in InstanceKey, see the provisioning step
						ServiceID:             os.Getenv("SERVICE_ID"),
						PlanID:                os.Getenv("PLAN_ID"),
						InstanceID:            os.Getenv("INSTANCE_ID"),
						Provisioned:           true,
						ProvisioningTriggered: false,
					},
					BindingID: os.Getenv("BINDING_ID"),
				},
			},
		},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
	}
	operationStorage.InsertDeprovisioningOperation(operation)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.JSONFormatter{})

	if len(operation.Cls.BindingID) > 0 {
		unbindStep := NewClsUnbindStep(clsConfig, operationStorage)
		op, _, err := unbindStep.Run(operation, log)
		require.NoError(t, err)
		operation = op
	}

	clsClient := cls.NewClient(clsConfig)
	clsDeprovisioner := cls.NewDeprovisioner(clsStorage, clsClient)

	step := NewClsDeprovisionStep(clsConfig, clsDeprovisioner, operationStorage)

	for i := 0; i < 10; i++ {
		op, offset, err := step.Run(operation, log)
		require.NoError(t, err)
		operation = op

		log.Debugf("CLS Instance: %#v\n", op.Cls.Instance)
		if !operation.Cls.Instance.Provisioned {
			require.Empty(t, op.Cls.Instance.InstanceID)
			break
		}

		time.Sleep(offset)
	}
}
