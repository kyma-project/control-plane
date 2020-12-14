// +build sm_integration

package provisioning

import (
	"encoding/json"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

// TestEmsSteps tests all EMS steps with real Service Manager
// Usage:
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestEmsSteps -count=1
func TestEmsProvisioningSteps(t *testing.T) {
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})

	offeringStep := NewServiceManagerOfferingStep("EMS_Offering",
		EmsOfferingName, EmsPlanName, func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
			return &op.Ems.Instance
		}, repo)

	provisioningStep := NewEmsProvisionStep(repo)
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
	ppBytes, _ := json.Marshal(pp)
	operation := internal.ProvisioningOperation{
		ProvisioningParameters: string(ppBytes),
		SMClientFactory:        cliFactory,
	}
	repo.InsertProvisioningOperation(operation)

	bindingStep := NewEmsBindStep(repo)

	log := logrus.New()

	operation, retry, err := offeringStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = provisioningStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = bindingStep.Run(operation, log)
	fmt.Printf(">>> %#v\n", operation.Ems)
	require.NoError(t, err)
//	require.Zero(t, retry)

	for i:=0; i < 30 ; i++ {  //wait 5 min
		time.Sleep(retry)
		operation, retry, err = bindingStep.Run(operation, log)
		fmt.Printf(">>> %#v\n", operation.Ems)
		require.NoError(t, err)
		if operation.Ems.BindingID != "" {
			break
		}
	}
	require.NoError(t, err)
	require.Zero(t, retry)

	require.NotEmpty(t, operation.Ems.Instance.InstanceID)
	require.NotEmpty(t, operation.Ems.BindingID)


	fmt.Printf("\nexport INSTANCE_ID=%s\nexport BINDING_ID=%s\n", operation.Ems.Instance.InstanceID, operation.Ems.BindingID)
}
