package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"

	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	ctrl "sigs.k8s.io/controller-runtime"

	restclient "k8s.io/client-go/rest"
)

const (
	databaseConnectionRetries = 20
	defaultSyncPeriod         = 10 * time.Minute
)

type DynamicKubeconfigProvider interface {
	FetchFromRequest(shootName string) ([]byte, error)
}

func newProvisioningService(
	gardenerProject string,
	provisioner provisioning.Provisioner,
	dbsFactory dbsession.Factory,
	shootProvider gardener.ShootProvider,
	provisioningQueue queue.OperationQueue,
	deprovisioningQueue queue.OperationQueue,
	shootUpgradeQueue queue.OperationQueue,
	defaultEnableKubernetesVersionAutoUpdate,
	defaultEnableMachineImageVersionAutoUpdate bool,
	defaultEnableIMDSv2 bool,
	dynamicKubeconfigProvider DynamicKubeconfigProvider) provisioning.Service {

	uuidGenerator := uuid.NewUUIDGenerator()
	inputConverter := provisioning.NewInputConverter(uuidGenerator, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate, defaultEnableIMDSv2)
	graphQLConverter := provisioning.NewGraphQLConverter()

	return provisioning.NewProvisioningService(
		inputConverter,
		graphQLConverter,
		dbsFactory,
		provisioner,
		uuidGenerator,
		shootProvider,
		provisioningQueue,
		deprovisioningQueue,
		shootUpgradeQueue,
		dynamicKubeconfigProvider)
}

func newShootController(gardenerNamespace string, gardenerClusterCfg *restclient.Config, dbsFactory dbsession.Factory, auditLogTenantConfigPath string) (*gardener.ShootController, error) {

	syncPeriod := defaultSyncPeriod

	mgr, err := ctrl.NewManager(gardenerClusterCfg, ctrl.Options{SyncPeriod: &syncPeriod, Namespace: gardenerNamespace})
	if err != nil {
		return nil, fmt.Errorf("unable to create shoot controller manager: %w", err)
	}

	return gardener.NewShootController(mgr, dbsFactory, auditLogTenantConfigPath)
}

func newGardenerClusterConfig(cfg config) (*restclient.Config, error) {
	rawKubeconfig, err := os.ReadFile(cfg.Gardener.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", cfg.Gardener.KubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := gardener.Config(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gardener cluster config: %s", err.Error())
	}

	return gardenerClusterConfig, nil
}
