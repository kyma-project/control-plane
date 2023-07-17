package queue

import (
	"context"
	"time"

	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/failure"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/hibernation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/shootupgrade"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/upgrade"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ProvisioningTimeouts struct {
	ClusterCreation        time.Duration `envconfig:"default=60m"`
	ClusterDomains         time.Duration `envconfig:"default=10m"`
	BindingsCreation       time.Duration `envconfig:"default=5m"`
	InstallationTriggering time.Duration `envconfig:"default=20m"`
	Installation           time.Duration `envconfig:"default=60m"`
	Upgrade                time.Duration `envconfig:"default=60m"`
	UpgradeTriggering      time.Duration `envconfig:"default=20m"`
	ShootUpgrade           time.Duration `envconfig:"default=30m"`
	ShootRefresh           time.Duration `envconfig:"default=5m"`
	AgentConfiguration     time.Duration `envconfig:"default=15m"`
	AgentConnection        time.Duration `envconfig:"default=15m"`
}

type ProvisioningNoInstallTimeouts struct {
	ClusterCreation    time.Duration `envconfig:"default=60m"`
	ClusterDomains     time.Duration `envconfig:"default=10m"`
	BindingsCreation   time.Duration `envconfig:"default=5m"`
	AgentConfiguration time.Duration `envconfig:"default=15m"`
}

type DeprovisioningNoInstallTimeouts struct {
	ClusterDeletion           time.Duration `envconfig:"default=30m"`
	WaitingForClusterDeletion time.Duration `envconfig:"default=60m"`
}

type DeprovisioningTimeouts struct {
	ClusterCleanup            time.Duration `envconfig:"default=20m"`
	ClusterDeletion           time.Duration `envconfig:"default=30m"`
	WaitingForClusterDeletion time.Duration `envconfig:"default=60m"`
}

type HibernationTimeouts struct {
	WaitingForClusterHibernation time.Duration `envconfig:"default=60m"`
}

type adminKubeconfig = func(context.Context, client.Object, client.Object, ...client.SubResourceCreateOption) error

func CreateProvisioningQueue(
	timeouts ProvisioningTimeouts,
	factory dbsession.Factory,
	installationClient installation.Service,
	configurator runtime.Configurator,
	ccClientConstructor provisioning.CompassConnectionClientConstructor,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
	secretsClient v1core.SecretInterface,
	request adminKubeconfig,
	operatorRoleBindingConfig provisioning.OperatorRoleBinding,
	k8sClientProvider k8s.K8sClientProvider) OperationQueue {

	waitForAgentToConnectStep := provisioning.NewWaitForAgentToConnectStep(ccClientConstructor, configurator, model.FinishedStage, timeouts.AgentConnection, directorClient)
	configureAgentStep := provisioning.NewConnectAgentStep(configurator, waitForAgentToConnectStep.Name(), timeouts.AgentConfiguration)
	waitForInstallStep := provisioning.NewWaitForInstallationStep(installationClient, configureAgentStep.Name(), timeouts.Installation, factory.NewWriteSession())
	installStep := provisioning.NewInstallKymaStep(installationClient, waitForInstallStep.Name(), timeouts.InstallationTriggering)
	createBindingsForOperatorsStep := provisioning.NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorRoleBindingConfig, installStep.Name(), timeouts.BindingsCreation)

	waitForClusterCreationStep := provisioning.NewWaitForClusterCreationStep(
		shootClient,
		factory.NewReadWriteSession(),
		gardener.NewKubeconfigProvider(secretsClient, request),
		createBindingsForOperatorsStep.Name(),
		timeouts.ClusterCreation)

	waitForClusterDomainStep := provisioning.NewWaitForClusterDomainStep(shootClient, directorClient, waitForClusterCreationStep.Name(), timeouts.ClusterDomains)

	provisionSteps := map[model.OperationStage]operations.Step{
		model.WaitForAgentToConnect:        waitForAgentToConnectStep,
		model.ConnectRuntimeAgent:          configureAgentStep,
		model.WaitingForInstallation:       waitForInstallStep,
		model.StartingInstallation:         installStep,
		model.CreatingBindingsForOperators: createBindingsForOperatorsStep,
		model.WaitingForClusterDomain:      waitForClusterDomainStep,
		model.WaitingForClusterCreation:    waitForClusterCreationStep,
	}

	provisioningExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.Provision,
		provisionSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(provisioningExecutor)
}

func CreateProvisioningNoInstallQueue(
	timeouts ProvisioningNoInstallTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
	secretsClient v1core.SecretInterface,
	request adminKubeconfig,
	operatorRoleBindingConfig provisioning.OperatorRoleBinding,
	k8sClientProvider k8s.K8sClientProvider,
	configurator runtime.Configurator) OperationQueue {

	configureAgentStep := provisioning.NewConnectAgentStep(configurator, model.FinishedStage, timeouts.AgentConfiguration)
	createBindingsForOperatorsStep := provisioning.NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorRoleBindingConfig, configureAgentStep.Name(), timeouts.BindingsCreation)
	waitForClusterCreationStep := provisioning.NewWaitForClusterCreationStep(
		shootClient,
		factory.NewReadWriteSession(),
		gardener.NewKubeconfigProvider(secretsClient, request),
		createBindingsForOperatorsStep.Name(),
		timeouts.ClusterCreation)

	waitForClusterDomainStep := provisioning.NewWaitForClusterDomainStep(shootClient, directorClient, waitForClusterCreationStep.Name(), timeouts.ClusterDomains)

	provisionNoInstallSteps := map[model.OperationStage]operations.Step{
		model.ConnectRuntimeAgent:          configureAgentStep,
		model.CreatingBindingsForOperators: createBindingsForOperatorsStep,
		model.WaitingForClusterDomain:      waitForClusterDomainStep,
		model.WaitingForClusterCreation:    waitForClusterCreationStep,
	}

	provisioningExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.ProvisionNoInstall,
		provisionNoInstallSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(provisioningExecutor)
}

func CreateUpgradeQueue(
	provisioningTimeouts ProvisioningTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	installationClient installation.Service) OperationQueue {

	updatingUpgradeStep := upgrade.NewUpdateUpgradeStateStep(factory.NewWriteSession(), model.FinishedStage, 5*time.Minute)
	waitForInstallStep := provisioning.NewWaitForInstallationStep(installationClient, updatingUpgradeStep.Name(), provisioningTimeouts.Installation, factory.NewWriteSession())
	upgradeStep := upgrade.NewUpgradeKymaStep(installationClient, waitForInstallStep.Name(), provisioningTimeouts.UpgradeTriggering)

	upgradeSteps := map[model.OperationStage]operations.Step{
		model.UpdatingUpgradeState:   updatingUpgradeStep,
		model.WaitingForInstallation: waitForInstallStep,
		model.StartingUpgrade:        upgradeStep,
	}

	upgradeExecutor := operations.NewExecutor(factory.NewReadWriteSession(),
		model.Upgrade,
		upgradeSteps,
		failure.NewUpgradeFailureHandler(factory.NewWriteSession()),
		directorClient,
	)

	return NewQueue(upgradeExecutor)
}

func CreateDeprovisioningQueue(
	timeouts DeprovisioningTimeouts,
	factory dbsession.Factory,
	installationClient installation.Service,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
	deleteDelay time.Duration) OperationQueue {

	waitForClusterDeletion := deprovisioning.NewWaitForClusterDeletionStep(shootClient, factory, directorClient, model.FinishedStage, timeouts.WaitingForClusterDeletion)
	deleteCluster := deprovisioning.NewDeleteClusterStep(shootClient, waitForClusterDeletion.Name(), timeouts.ClusterDeletion)
	triggerKymaUninstall := deprovisioning.NewTriggerKymaUninstallStep(shootClient, installationClient, deleteCluster.Name(), 5*time.Minute, deleteDelay)
	cleanupCluster := deprovisioning.NewCleanupClusterStep(shootClient, installationClient, triggerKymaUninstall.Name(), timeouts.ClusterCleanup)

	deprovisioningSteps := map[model.OperationStage]operations.Step{
		model.CleanupCluster:         cleanupCluster,
		model.DeleteCluster:          deleteCluster,
		model.WaitForClusterDeletion: waitForClusterDeletion,
		model.TriggerKymaUninstall:   triggerKymaUninstall,
	}

	deprovisioningExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.Deprovision,
		deprovisioningSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(deprovisioningExecutor)
}

func CreateDeprovisioningNoInstallQueue(
	timeouts DeprovisioningNoInstallTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
) OperationQueue {

	waitForClusterDeletion := deprovisioning.NewWaitForClusterDeletionStep(shootClient, factory, directorClient, model.FinishedStage, timeouts.WaitingForClusterDeletion)
	deleteCluster := deprovisioning.NewDeleteClusterStep(shootClient, waitForClusterDeletion.Name(), timeouts.ClusterDeletion)

	deprovisioningNoInstallSteps := map[model.OperationStage]operations.Step{
		model.DeleteCluster:          deleteCluster,
		model.WaitForClusterDeletion: waitForClusterDeletion,
	}

	deprovisioningExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.DeprovisionNoInstall,
		deprovisioningNoInstallSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(deprovisioningExecutor)
}

func CreateShootUpgradeQueue(
	timeouts ProvisioningTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
	request adminKubeconfig,
	operatorRoleBindingConfig provisioning.OperatorRoleBinding,
	k8sClientProvider k8s.K8sClientProvider,
	secretsClient v1core.SecretInterface,
) OperationQueue {

	createBindingsForOperatorsStep := provisioning.NewCreateBindingsForOperatorsStep(
		k8sClientProvider,
		operatorRoleBindingConfig,
		model.FinishedStage,
		timeouts.BindingsCreation)

	provider := gardener.NewKubeconfigProvider(secretsClient, request)

	waitForShootUpgrade := shootupgrade.NewWaitForShootUpgradeStep(
		shootClient,
		factory.NewReadWriteSession(),
		provider,
		createBindingsForOperatorsStep.Name(),
		timeouts.ShootUpgrade)

	waitForShootNewVersion := shootupgrade.NewWaitForShootNewVersionStep(shootClient, waitForShootUpgrade.Name(), timeouts.ShootRefresh)

	upgradeSteps := map[model.OperationStage]operations.Step{
		model.CreatingBindingsForOperators: createBindingsForOperatorsStep,
		model.WaitingForShootUpgrade:       waitForShootUpgrade,
		model.WaitingForShootNewVersion:    waitForShootNewVersion,
	}

	upgradeClusterExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.UpgradeShoot,
		upgradeSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(upgradeClusterExecutor)
}

func CreateHibernationQueue(
	timeouts HibernationTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface) OperationQueue {

	waitForHibernation := hibernation.NewWaitForHibernationStep(shootClient, model.FinishedStage, timeouts.WaitingForClusterHibernation)

	hibernationSteps := map[model.OperationStage]operations.Step{
		model.WaitForHibernation: waitForHibernation,
	}

	hibernateClusterExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.Hibernate,
		hibernationSteps,
		failure.NewNoopFailureHandler(),
		directorClient,
	)

	return NewQueue(hibernateClusterExecutor)
}
