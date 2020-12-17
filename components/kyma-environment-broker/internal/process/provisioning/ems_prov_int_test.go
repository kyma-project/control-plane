// +build sm_integration

package provisioning

import (
	"encoding/json"
	"fmt"
<<<<<<< HEAD
<<<<<<< HEAD
=======
>>>>>>> ec1e40a0... Solve check-imports issues
	"os"
	"testing"
	"time"

<<<<<<< HEAD
=======
>>>>>>> 7b4ea82d... Add int tests
=======
>>>>>>> ec1e40a0... Solve check-imports issues
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
<<<<<<< HEAD
<<<<<<< HEAD
=======
	"os"
	"testing"
	"time"
>>>>>>> 7b4ea82d... Add int tests
=======
>>>>>>> ec1e40a0... Solve check-imports issues
)

// TestEmsSteps tests all EMS steps with real Service Manager
// Usage:
<<<<<<< HEAD
// export SM_USERNAME=+9/MttPOR3JTo2LLYOYR/WkWa1T72pyuhWiQJB3ieIk=
// export SM_PASSWORD=p+gmv1gcsN1V3qqZpugwH5aru5sp9my2fsTTdido20o=
// export SM_URL=https://service-manager.cfapps.sap.hana.ondemand.com
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestEmsSteps -count=1
func TestEmsProvisioningSteps(t *testing.T) {
=======
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestEmsSteps -count=1
<<<<<<< HEAD
<<<<<<< HEAD
func TestEmsSteps(t *testing.T) {
>>>>>>> 7b4ea82d... Add int tests
=======
=======
const (
	secretKey = "1234567890123456"
)

>>>>>>> 89a5fd0f... Persist EMS overrides in DB
func TestEmsProvisioningSteps(t *testing.T) {
>>>>>>> 3ac83ef0... Update integration tests
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
	simpleInputCreator := newInputCreator()
	operation.InputCreator = simpleInputCreator

	repo.InsertProvisioningOperation(operation)

	bindingStep := NewEmsBindStep(repo, secretKey)

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
<<<<<<< HEAD
<<<<<<< HEAD
	//	require.Zero(t, retry)

	for i := 0; i < 30; i++ { //wait 5 min
=======
//	require.Zero(t, retry)

	for i:=0; i < 30 ; i++ {  //wait 5 min
>>>>>>> 7b4ea82d... Add int tests
=======
	//	require.Zero(t, retry)

	for i := 0; i < 30; i++ { //wait 5 min
>>>>>>> ec1e40a0... Solve check-imports issues
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

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
=======

>>>>>>> 7b4ea82d... Add int tests
=======
>>>>>>> 96ead07e... Add eventing overrides
=======
	overridesOut, err := decryptOverrides(secretKey, operation.Ems.Overrides)
	require.NoError(t, err)

>>>>>>> 89a5fd0f... Persist EMS overrides in DB
	fmt.Printf("\nexport INSTANCE_ID=%s\nexport BINDING_ID=%s\n", operation.Ems.Instance.InstanceID, operation.Ems.BindingID)
	fmt.Printf("\nexport OVERRIDES=%#v\n", overridesOut)
}
