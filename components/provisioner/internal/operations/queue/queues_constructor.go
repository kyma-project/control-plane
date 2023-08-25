package queue

import (
	"time"

	gardener_apis "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/failure"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/shootupgrade"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
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

type DeprovisioningTimeouts struct {
	ClusterDeletion           time.Duration `envconfig:"default=30m"`
	WaitingForClusterDeletion time.Duration `envconfig:"default=60m"`
}

type HibernationTimeouts struct {
	WaitingForClusterHibernation time.Duration `envconfig:"default=60m"`
}

//go:generate mockery --name=KubeconfigProvider
type KubeconfigProvider interface {
	FetchFromShoot(shootName string) ([]byte, error)
	FetchFromRequest(shootName string) ([]byte, error)
}

func CreateProvisioningQueue(
	timeouts ProvisioningTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
	operatorRoleBindingConfig provisioning.OperatorRoleBinding,
	k8sClientProvider k8s.K8sClientProvider,
	configurator runtime.Configurator,
	kubeconfigProvider KubeconfigProvider) OperationQueue {

	configureAgentStep := provisioning.NewConnectAgentStep(configurator, kubeconfigProvider, model.FinishedStage, timeouts.AgentConfiguration)
	createBindingsForOperatorsStep := provisioning.NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorRoleBindingConfig, kubeconfigProvider, configureAgentStep.Name(), timeouts.BindingsCreation)
	waitForClusterCreationStep := provisioning.NewWaitForClusterCreationStep(shootClient, factory.NewReadWriteSession(), kubeconfigProvider, createBindingsForOperatorsStep.Name(), timeouts.ClusterCreation)
	waitForClusterDomainStep := provisioning.NewWaitForClusterDomainStep(shootClient, directorClient, waitForClusterCreationStep.Name(), timeouts.ClusterDomains)

	provisionSteps := map[model.OperationStage]operations.Step{
		model.ConnectRuntimeAgent:          configureAgentStep,
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

func CreateDeprovisioningQueue(
	timeouts DeprovisioningTimeouts,
	factory dbsession.Factory,
	directorClient director.DirectorClient,
	shootClient gardener_apis.ShootInterface,
) OperationQueue {

	waitForClusterDeletion := deprovisioning.NewWaitForClusterDeletionStep(shootClient, factory, directorClient, model.FinishedStage, timeouts.WaitingForClusterDeletion)
	deleteCluster := deprovisioning.NewDeleteClusterStep(shootClient, waitForClusterDeletion.Name(), timeouts.ClusterDeletion)

	deprovisioningSteps := map[model.OperationStage]operations.Step{
		model.DeleteCluster:          deleteCluster,
		model.WaitForClusterDeletion: waitForClusterDeletion,
	}

	deprovisioningExecutor := operations.NewExecutor(
		factory.NewReadWriteSession(),
		model.DeprovisionNoInstall,
		deprovisioningSteps,
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
	operatorRoleBindingConfig provisioning.OperatorRoleBinding,
	k8sClientProvider k8s.K8sClientProvider,
	secretsClient v1core.SecretInterface,
) OperationQueue {

	kubeconfigProvider := gardener.NewKubeconfigProvider(nil, nil, secretsClient)
	createBindingsForOperatorsStep := provisioning.NewCreateBindingsForOperatorsStep(k8sClientProvider, operatorRoleBindingConfig, nil, model.FinishedStage, timeouts.BindingsCreation)
	waitForShootUpgrade := shootupgrade.NewWaitForShootUpgradeStep(shootClient, factory.NewReadWriteSession(), kubeconfigProvider, createBindingsForOperatorsStep.Name(), timeouts.ShootUpgrade)
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
