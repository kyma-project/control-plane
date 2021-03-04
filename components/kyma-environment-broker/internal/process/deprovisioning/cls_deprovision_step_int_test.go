// +build sm_integration

package deprovisioning

import (
	"fmt"
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

	instance := internal.NewCLSInstance(globalAccountID, "eu", internal.WithReferences(skrInstanceID))
	db := storage.NewMemoryStorage()
	db.CLSInstances().Insert(*instance)

	repo := db.Operations()

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceID: skrInstanceID,
			ProvisioningParameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{GlobalAccountID: globalAccountID}},
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Region: "eu",
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:              os.Getenv("BROKER_ID"), // saved in InstanceKey, see the provisioning step
						ServiceID:             os.Getenv("SERVICE_ID"),
						PlanID:                os.Getenv("PLAN_ID"),
						InstanceID:            os.Getenv("INSTANCE_ID"),
						Provisioned:           true,
						ProvisioningTriggered: false,
					},
				},
			},
		},
		SMClientFactory: servicemanager.NewClientFactory(servicemanager.Config{}),
	}

	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	clsClient := cls.NewClient(clsConfig, log)
	clsDeprovisioner := cls.NewDeprovisioner(db.CLSInstances(), clsClient, log)

	step := NewClsDeprovisionStep(clsConfig, repo, clsDeprovisioner)

	for i := 0; i < 10; i++ {
		op, offset, err := step.Run(operation, log)
		require.NoError(t, err)
		operation = op

		fmt.Printf("deprovisioned flag: %#v", op.Cls.Instance)
		if !operation.Cls.Instance.Provisioned {
			require.Empty(t, op.Cls.Instance.InstanceID)
			break
		}

		time.Sleep(offset)
	}
}
