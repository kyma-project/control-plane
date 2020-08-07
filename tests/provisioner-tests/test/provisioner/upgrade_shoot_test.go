package provisioner

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/assertions"
	"github.com/sirupsen/logrus"
)

func TestShootUpgrade(t *testing.T) {
	t.Parallel()

	globalLog := logrus.WithField("TestId", testSuite.TestId)

	globalLog.Infof("Starting Kyma Control Plane Runtime Provisioner tests of Shoot Upgrade on Gardener")
	wg := &sync.WaitGroup{}

	for _, provider := range testSuite.gardenerProviders {
		wg.Add(1)
		go func(provider string) {
			defer wg.Done()
			defer testSuite.Recover()

			t.Run(provider, func(t *testing.T) {
				log := NewLogger(t, fmt.Sprintf("Provider=%s", provider))

				// Create provisioning input
				provisioningInput, err := testkit.CreateGardenerProvisioningInput(&testSuite.config, testSuite.config.Kyma.Version, provider)
				assertions.RequireNoError(t, err)

				runtimeName := fmt.Sprintf("provisioner-upgr-sh-test-%s-%s", strings.ToLower(provider), uuid.New().String()[:4])
				provisioningInput.RuntimeInput.Name = runtimeName

				// Provision runtime
				log.Log("Starting provisioning...")
				provisioningOperationID, runtimeID, err := testSuite.ProvisionerClient.ProvisionRuntime(provisioningInput)
				assertions.RequireNoError(t, err, "Error while starting Runtime provisioning")
				defer ensureClusterIsDeprovisioned(runtimeID)

				log.AddField(fmt.Sprintf("RuntimeId=%s", runtimeID))
				log.AddField(fmt.Sprintf("ProvisioningOperationId=%s", provisioningOperationID))

				// Wait for provisioning to finish
				log.Log("Waiting for provisioning to finish...")
				provisioningOperationStatus, err := testSuite.WaitUntilOperationIsFinished(ProvisioningTimeout, provisioningOperationID)
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

				upgradeShootConfig := testkit.CreateGardenerUpgradeInput(&testSuite.config, provider)
				log.Log("Starting shoot upgrade...")

				upgradeOperationStatus, err := testSuite.ProvisionerClient.UpgradeShoot(runtimeID, *upgradeShootConfig)

				upgradeOperationStatus, err = testSuite.WaitUntilOperationIsFinished(UpgradeShootTimeout, *upgradeOperationStatus.ID)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeUpgradeShoot, runtimeID, upgradeOperationStatus)
				log.Log("Shoot upgrade finished.")

				log.Log("Accessing API Server after upgrade...")
				_, err = k8sClient.ServerVersion()
				assertions.RequireNoError(t, err)

				// Fetch Runtime Status
				log.Log("Getting Runtime status...")
				runtimeStatus, err = testSuite.ProvisionerClient.RuntimeStatus(runtimeID)
				assertions.RequireNoError(t, err)

				log.Log("Asserting expected runtime state...")
				assertions.AssertUpgradedClusterState(t, *upgradeShootConfig.GardenerConfig, *runtimeStatus.RuntimeConfiguration.ClusterConfig)

				// Deprovisioning runtime
				log.Log("Starting Runtime deprovisioning...")
				deprovisioningOperationID, err := testSuite.ProvisionerClient.DeprovisionRuntime(runtimeID)
				assertions.RequireNoError(t, err)

				log.AddField(fmt.Sprintf("DeprovisioningOperationId=%s", deprovisioningOperationID))

				// Get deprovisioning Operation Status
				deprovisioningOperationStatus, err := testSuite.ProvisionerClient.RuntimeOperationStatus(deprovisioningOperationID)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationInProgress(t, gqlschema.OperationTypeDeprovision, runtimeID, deprovisioningOperationStatus)

				log.Log("Waiting for deprovisioning to finish...")
				deprovisioningOperationStatus, err = testSuite.WaitUntilOperationIsFinished(DeprovisioningTimeout, deprovisioningOperationID)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeDeprovision, runtimeID, deprovisioningOperationStatus)
				log.Log("Deprovisioning finished.")
			})
		}(provider)
	}
	wg.Wait()

}
