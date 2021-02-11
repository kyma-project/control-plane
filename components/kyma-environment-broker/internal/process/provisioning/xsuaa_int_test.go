// +build sm_integration

package provisioning

import (
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	uaa "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/xsuaa"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestProvisioningSteps tests all XSUAA steps with real Service Manager
// Usage:
// export SM_USERNAME=
// export SM_PASSWORD=
// export SM_URL=
// go test -v -tags=sm_integration ./internal/process/provisioning/... -run TestProvisioningSteps -count=1
func TestProvisioningSteps(t *testing.T) {
	repo := storage.NewMemoryStorage().Operations()
	cliFactory := servicemanager.NewClientFactory(servicemanager.Config{
		OverrideMode: servicemanager.SMOverrideModeNever,
		URL:          "",
		Password:     "",
		Username:     "",
	})

	offeringStep := NewServiceManagerOfferingStep("XSUAA_Offering",
		"xsuaa", "application", func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
			return &op.XSUAA.Instance
		}, repo)

	provisioningStep := NewXSUAAProvisioningStep(repo, uaa.Config{
		DeveloperGroup:      "devGroup",
		DeveloperRole:       "devRole",
		NamespaceAdminGroup: "nag",
		NamespaceAdminRole:  "nar",
	})
	bindingStep := NewXSUAABindingStep(repo)

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
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				ShootDomain: "uaa-test.kyma-dev.shoot.canary.k8s-hana.ondemand.com",
			},
			ProvisioningParameters: pp,
		},
		SMClientFactory: cliFactory,
	}
	repo.InsertProvisioningOperation(operation)
	log := logrus.New()

	operation, retry, err := offeringStep.Run(operation, log)
	fmt.Printf(">>> %+v\n", operation.XSUAA)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, _, _ = provisioningStep.Run(operation, log)
	fmt.Printf(">>> %+v\n", operation.XSUAA)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, _, _ = bindingStep.Run(operation, log)
	fmt.Printf(">>> %+v\n", operation.XSUAA)
	require.NoError(t, err)
	require.Zero(t, retry)

	fmt.Printf("\nexport INSTANCE_ID=%s\nexport BINDING_ID=%s\n", operation.XSUAA.Instance.InstanceID, operation.XSUAA.BindingID)

}
