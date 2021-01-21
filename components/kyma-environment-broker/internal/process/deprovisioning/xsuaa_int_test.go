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

// TestProvisioningSteps tests all XSUAA steps with real Service Manager
// Usage:
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// export INSTANCE_ID=
// export BINDING_ID=
// export BROKER_ID=b122df80-b1ea-44d2-839f-3d17199a5b78 #for staging
// export BROKER_ID=fb8e7037-0c56-405f-8110-187ef9d39273 #for canary
// export SERVICE_ID=xsuaa
// export PLAN_ID=ThGdx5loQ6XhvcdY6dLlEXcTgQD7641pDKXJfzwYGLg=
// go test -v -tags=sm_integration ./internal/process/deprovisioning/... -run TestDeprovisioningSteps -count=1
func TestDeprovisioningSteps(t *testing.T) {
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})
	unbindingStep := NewXSUAAUnbindStep(repo)
	deprovisioningStep := NewXSUAADeprovisionStep(repo)
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
				XSUAA: internal.XSUAAData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:              os.Getenv("BROKER_ID"),
						ServiceID:             os.Getenv("SERVICE_ID"),
						PlanID:                os.Getenv("PLAN_ID"),
						InstanceID:            os.Getenv("INSTANCE_ID"),
						Provisioned:           true,
						ProvisioningTriggered: true,
					},
					XSAppname: "",
					BindingID: os.Getenv("BINDING_ID"),
				},
			},
			ProvisioningParameters: pp,
		},
		SMClientFactory: cliFactory,
	}
	repo.InsertDeprovisioningOperation(operation)
	log := logrus.New()

	operation, retry, err := unbindingStep.Run(operation, log)
	fmt.Printf(">>> %+v\n", operation.XSUAA)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, _, _ = deprovisioningStep.Run(operation, log)
	fmt.Printf(">>> %+v\n", operation.XSUAA)
	require.NoError(t, err)
	require.Zero(t, retry)
}
