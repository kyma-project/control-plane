// +build sm_integration

package deprovisioning

import (
	"encoding/json"
	"fmt"
<<<<<<< HEAD
<<<<<<< HEAD
	"os"
	"testing"

=======
>>>>>>> 7b4ea82d... Add int tests
=======
	"os"
	"testing"

>>>>>>> ec1e40a0... Solve check-imports issues
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
<<<<<<< HEAD
<<<<<<< HEAD
)

// TestProvisioningSteps tests all XSUAA steps with real Service Manager
// Usage:
// export SM_USERNAME=+9/MttPOR3JTo2LLYOYR/WkWa1T72pyuhWiQJB3ieIk=
// export SM_PASSWORD=p+gmv1gcsN1V3qqZpugwH5aru5sp9my2fsTTdido20o=
// export SM_URL=https://service-manager.cfapps.sap.hana.ondemand.com
// export INSTANCE_ID=
// export BINDING_ID=
// export BROKER_ID=61e5fdc2-40e6-4cd6-b0e0-27848372113e
// export SERVICE_ID=588f637e-5de8-4b60-8fda-8d9015c55052
// export PLAN_ID=c267fe4e-de8e-44b6-825b-9d0cf233b318
// go test -v -tags=sm_integration ./internal/process/deprovisioning/... -run TestDeprovisioningSteps
func TestEmsDeprovisioningSteps(t *testing.T) {
=======
	"os"
	"testing"
=======
>>>>>>> ec1e40a0... Solve check-imports issues
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
<<<<<<< HEAD
func TestDeprovisioningSteps(t *testing.T) {
>>>>>>> 7b4ea82d... Add int tests
=======
func TestEmsDeprovisioningSteps(t *testing.T) {
>>>>>>> 3ac83ef0... Update integration tests
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
<<<<<<< HEAD
<<<<<<< HEAD
			ServiceManager: &internal.ServiceManagerEntryDTO{
=======
			ServiceManager: &internal.ServiceManagerEntryDTO {
>>>>>>> 7b4ea82d... Add int tests
=======
			ServiceManager: &internal.ServiceManagerEntryDTO{
>>>>>>> ec1e40a0... Solve check-imports issues
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
	ppBytes, _ := json.Marshal(pp)
<<<<<<< HEAD
<<<<<<< HEAD
	operation := internal.DeprovisioningOperation{
		ProvisioningParameters: string(ppBytes),
		SMClientFactory:        cliFactory,
		Ems: internal.EmsData{
			Instance: internal.ServiceManagerInstanceInfo{
				BrokerID:    os.Getenv("BROKER_ID"), // saved in InstanceKey, see the provisioning step
				ServiceID:   os.Getenv("SERVICE_ID"),
				PlanID:      os.Getenv("PLAN_ID"),
				InstanceID:  os.Getenv("INSTANCE_ID"),
				Provisioned: true,
=======
	operation := internal.DeprovisioningOperation {
=======
	operation := internal.DeprovisioningOperation{
>>>>>>> ec1e40a0... Solve check-imports issues
		ProvisioningParameters: string(ppBytes),
		SMClientFactory:        cliFactory,
		Ems: internal.EmsData{
			Instance: internal.ServiceManagerInstanceInfo{
				BrokerID:              os.Getenv("BROKER_ID"), // saved in InstanceKey, see the provisioning step
				ServiceID:             os.Getenv("SERVICE_ID"),
				PlanID:                os.Getenv("PLAN_ID"),
				InstanceID:            os.Getenv("INSTANCE_ID"),
				Provisioned:           true,
<<<<<<< HEAD
>>>>>>> 7b4ea82d... Add int tests
				//ProvisioningTriggered: true,
=======
				ProvisioningTriggered: false,
>>>>>>> 3ac83ef0... Update integration tests
			},
			BindingID: os.Getenv("BINDING_ID"),
		},
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
<<<<<<< HEAD
<<<<<<< HEAD
}
=======
}
>>>>>>> 7b4ea82d... Add int tests
=======
}
>>>>>>> ec1e40a0... Solve check-imports issues
