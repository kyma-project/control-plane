package provisioner

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/assertions"
)

func TestRuntimeUpgrade(t *testing.T) {
	t.Parallel()

	globalLog := logrus.WithField("TestID", testSuite.TestID)

	globalLog.Infof("Starting Kyma Control Plane Runtime Provisioner tests of Runtime Upgrade on Gardener")
	wg := &sync.WaitGroup{}

	for _, provider := range testSuite.gardenerProviders {
		wg.Add(1)
		go func(provider string) {
			defer wg.Done()
			defer testSuite.Recover()

			t.Run(provider, func(t *testing.T) {
				log := testkit.NewLogger(t, logrus.Fields{
					"Provider": provider,
					"TestType": "upgrade-runtime",
				})

				// Create provisioning input
				provisioningInput, err := testkit.CreateGardenerProvisioningInput(&testSuite.config, testSuite.config.Kyma.PreUpgradeVersion, provider)
				assertions.RequireNoError(t, err)

				runtimeName := fmt.Sprintf("provisioner-upgrade-test-%s-%s", strings.ToLower(provider), uuid.New().String()[:4])
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
				provisioningOperationStatus, err := testSuite.WaitUntilOperationIsFinished(testSuite.config.Timeouts.ProvisioningTimeout, provisioningOperationID, log)
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

				// TODO: To properly test upgrade of specific components we should setup some resources on cluster here

				upgradedKymaConfig, err := testkit.CreateKymaConfigInput(testSuite.config.Kyma.Version)
				upgradeRuntimeInput := gqlschema.UpgradeRuntimeInput{KymaConfig: upgradedKymaConfig}

				log.Log("Starting upgrade...")
				upgradeOperationStatus, err := testSuite.ProvisionerClient.UpgradeRuntime(runtimeID, upgradeRuntimeInput)
				assertions.RequireNoError(t, err, "Error while starting Runtime upgrade")
				require.NotNil(t, upgradeOperationStatus.ID)

				log.WithField("UpgradeOperationID", *upgradeOperationStatus.ID)

				log.Log("Waiting for upgrade to finish...")
				upgradeOperationStatus, err = testSuite.WaitUntilOperationIsFinished(testSuite.config.Timeouts.UpgradeTimeout, *upgradeOperationStatus.ID, log)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeUpgrade, runtimeID, upgradeOperationStatus)
				log.Log("Upgrade finished.")

				log.Log("Accessing API Server after upgrade...")
				_, err = k8sClient.ServerVersion()
				assertions.RequireNoError(t, err)

				// TODO: To properly test is components are upgraded some tests should be run on cluster

				// Deprovisioning runtime
				log.Log("Starting Runtime deprovisioning...")
				deprovisioningOperationID, err := testSuite.ProvisionerClient.DeprovisionRuntime(runtimeID)
				assertions.RequireNoError(t, err)

				log.WithField("DeprovisioningOperationID", deprovisioningOperationID)

				// Get provisioning Operation Status
				deprovisioningOperationStatus, err := testSuite.ProvisionerClient.RuntimeOperationStatus(deprovisioningOperationID)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationInProgress(t, gqlschema.OperationTypeDeprovision, runtimeID, deprovisioningOperationStatus)

				log.Log("Waiting for deprovisioning to finish...")
				deprovisioningOperationStatus, err = testSuite.WaitUntilOperationIsFinished(testSuite.config.Timeouts.DeprovisioningTimeout, deprovisioningOperationID, log)
				assertions.RequireNoError(t, err)
				assertions.AssertOperationSucceed(t, gqlschema.OperationTypeDeprovision, runtimeID, deprovisioningOperationStatus)
				log.Log("Deprovisioning finished.")
			})
		}(provider)
	}
	wg.Wait()

}
