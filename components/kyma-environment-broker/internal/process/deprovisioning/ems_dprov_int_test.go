// +build sm_integration

package deprovisioning

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
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
func TestEmsDeprovisioningSteps(t *testing.T) {
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})

	unbindingStep := NewEmsUnbindStep(repo)

	deprovisioningStep := NewEmsDeprovisionStep(repo)
	pp := internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			ServiceManager: &internal.ServiceManagerEntryDTO{
				URL: os.Getenv("SM_URL"),
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: os.Getenv("SM_USERNAME"),
						Password: os.Getenv("SM_PASSWORD"),
					},
				},
			},
		},
	}
	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Ems: internal.EmsData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:              os.Getenv("BROKER_ID"), // saved in InstanceKey, see the provisioning step
						ServiceID:             os.Getenv("SERVICE_ID"),
						PlanID:                os.Getenv("PLAN_ID"),
						InstanceID:            os.Getenv("INSTANCE_ID"),
						Provisioned:           true,
						ProvisioningTriggered: false,
					},
					BindingID: os.Getenv("BINDING_ID"),
					Overrides: "encryptedEventingOverrides",
				},
			},
			ProvisioningParameters: pp,
		},
		SMClientFactory: cliFactory,
	}
	repo.InsertDeprovisioningOperation(operation)

	log := logrus.New()

	operation, retry, err := unbindingStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = deprovisioningStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)
}
