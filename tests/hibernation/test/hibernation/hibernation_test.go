package provisioner

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/control-plane/tests/hibernation/test/testkit"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/assertions"
)

func TestShootUpgrade(t *testing.T) {
	t.Parallel()

	globalLog := logrus.WithField("TestID", testSuite.TestID)

	globalLog.Infof("Starting Kyma Control Plane Runtime Provisioner tests of Shoot Upgrade on Gardener")
	wg := &sync.WaitGroup{}

	for _, provider := range testSuite.gardenerProviders {
		wg.Add(1)
		go func(provider string) {
			defer wg.Done()
			defer testSuite.Recover()

			t.Run(provider, func(t *testing.T) {
				log := testkit.NewLogger(t, logrus.Fields{
					"Provider": provider,
					"TestType": "upgrade-shoot",
				})

				// Provisioning runtime
				// Create provisioning input
				provisioningInput, err := testkit.CreateGardenerProvisioningInput(&testSuite.config, testSuite.config.Kyma.Version, provider)
				assertions.RequireNoError(t, err)

				runtimeName := fmt.Sprintf("provisioner-test-%s-%s", strings.ToLower(provider), uuid.New().String()[:4])
				provisioningInput.RuntimeInput.Name = runtimeName

				// Provision runtime
				log.Log("Starting provisioning...")
				provisioningOperationID, runtimeID, err := testSuite.ProvisionerClient.ProvisionRuntime(provisioningInput)
				assertions.RequireNoError(t, err, "Error while starting Runtime provisioning")
				defer ensureClusterIsDeprovisioned(runtimeID, log)

				log.WithField("RuntimeID", runtimeID)
				log.WithField("ProvisioningOperationID", provisioningOperationID)

				// Wait for provisioning to finish
				log.Log("Waiting for provisioning to finish...")
				provisioningOperationStatus, err := testSuite.WaitUntilOperationIsFinished(ProvisioningTimeout, provisioningOperationID, log)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeProvision, runtimeID, provisioningOperationStatus)
				log.Log("Provisioning finished.")

				// Fetch Runtime Status
				log.Log("Getting Runtime status...")
				runtimeStatus, err := testSuite.ProvisionerClient.RuntimeStatus(runtimeID)
				assertions.RequireNoError(t, err)

				log.Log("Preparing K8s client...")
				k8sClient := testSuite.KubernetesClientFromRawConfig(t, *runtimeStatus.RuntimeConfiguration.Kubeconfig)

				log.Log("Accessing API Server on provisioned cluster...")
				_, err = k8sClient.ServerVersion()
				assertions.RequireNoError(t, err)

				// Hibernate runtime
				hibernationOperationID, err := testSuite.ProvisionerClient.HibernateRuntime(runtimeID)
				assertions.RequireNoError(t, err)

				log.Log("Waiting for hibernation to finish...")
				hibernationOperationStatus, err := testSuite.WaitUntilOperationIsFinished(HibernationTimeout, hibernationOperationID, log)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeDeprovision, runtimeID, hibernationOperationStatus)

				// Deprovisioning runtime
				log.Log("Starting Runtime deprovisioning...")
				deprovisioningOperationID, err := testSuite.ProvisionerClient.DeprovisionRuntime(runtimeID)
				assertions.RequireNoError(t, err)

				log.WithField("DeprovisioningOperationID", deprovisioningOperationID)

				// Get deprovisioning Operation Status
				deprovisioningOperationStatus, err := testSuite.ProvisionerClient.RuntimeOperationStatus(deprovisioningOperationID)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationInProgress(t, gqlschema.OperationTypeDeprovision, runtimeID, deprovisioningOperationStatus)

				log.Log("Waiting for deprovisioning to finish...")
				deprovisioningOperationStatus, err = testSuite.WaitUntilOperationIsFinished(DeprovisioningTimeout, deprovisioningOperationID, log)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeHibernate, runtimeID, deprovisioningOperationStatus)
				log.Log("Deprovisioning finished.")
			})
		}(provider)
	}
	wg.Wait()

}

func ensureClusterIsDeprovisioned(runtimeId string, log *testkit.Logger) {
	log.Logf("Ensuring the cluster is deprovisioned...")
	deprovisioningOperationId, err := testSuite.ProvisionerClient.DeprovisionRuntime(runtimeId)
	if err != nil {
		log.Errorf("Ensuring the cluster is deprovisioned failed, cluster might have already been deprovisioned: %s", err.Error())
		return
	}

	log.Logf("Deprovisioning operation id: %s", deprovisioningOperationId)
	deprovisioningOperationStatus, err := testSuite.WaitUntilOperationIsFinished(DeprovisioningTimeout, deprovisioningOperationId, log)
	if err != nil {
		log.Errorf("Error while waiting for deprovisioning operation to finish: %s", err.Error())
		return
	}

	if deprovisioningOperationStatus.State != gqlschema.OperationStateSucceeded {
		log.Errorf("Ensuring the cluster is deprovisioned failed with operation status %s with message %s", deprovisioningOperationStatus.State, unwrapString(deprovisioningOperationStatus.Message))
	}
}

func unwrapString(str *string) string {
	if str != nil {
		return *str
	}

	return ""
}
