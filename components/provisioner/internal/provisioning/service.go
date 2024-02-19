package provisioning

import (
	"time"

	gardener_Types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/hashicorp/go-version"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	uuid "github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	log "github.com/sirupsen/logrus"
)

type DynamicKubeconfigProvider interface {
	FetchFromRequest(shootName string) ([]byte, error)
}

//go:generate mockery --name=Service
type Service interface {
	ProvisionRuntime(config gqlschema.ProvisionRuntimeInput, tenant, subAccount string) (*gqlschema.OperationStatus, apperrors.AppError)
	DeprovisionRuntime(id string) (string, apperrors.AppError)
	UpgradeGardenerShoot(id string, input gqlschema.UpgradeShootInput) (*gqlschema.OperationStatus, apperrors.AppError)
	ReconnectRuntimeAgent(id string) (string, apperrors.AppError)
	RuntimeStatus(id string) (*gqlschema.RuntimeStatus, apperrors.AppError)
	RuntimeOperationStatus(id string) (*gqlschema.OperationStatus, apperrors.AppError)
}

//go:generate mockery --name=Provisioner
type Provisioner interface {
	ProvisionCluster(cluster model.Cluster, operationId string) apperrors.AppError
	DeprovisionCluster(cluster model.Cluster, operationId string) (model.Operation, apperrors.AppError)
	UpgradeCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError
}

//go:generate mockery --name=ShootProvider
type ShootProvider interface {
	Get(runtimeID string, tenant string) (gardener_Types.Shoot, apperrors.AppError)
}

type service struct {
	inputConverter            InputConverter
	graphQLConverter          GraphQLConverter
	directorService           director.DirectorClient
	shootProvider             ShootProvider
	dynamicKubeconfigProvider DynamicKubeconfigProvider

	dbSessionFactory dbsession.Factory
	provisioner      Provisioner
	uuidGenerator    uuid.UUIDGenerator

	provisioningQueue   queue.OperationQueue
	deprovisioningQueue queue.OperationQueue
	upgradeQueue        queue.OperationQueue
	shootUpgradeQueue   queue.OperationQueue
	hibernationQueue    queue.OperationQueue
}

func NewProvisioningService(
	inputConverter InputConverter,
	graphQLConverter GraphQLConverter,
	directorService director.DirectorClient,
	factory dbsession.Factory,
	provisioner Provisioner,
	generator uuid.UUIDGenerator,
	shootProvider ShootProvider,
	provisioningQueue queue.OperationQueue,
	deprovisioningQueue queue.OperationQueue,
	shootUpgradeQueue queue.OperationQueue,
	dynamicKubeconfigProvider DynamicKubeconfigProvider,

) Service {
	return &service{
		inputConverter:            inputConverter,
		graphQLConverter:          graphQLConverter,
		directorService:           directorService,
		dbSessionFactory:          factory,
		provisioner:               provisioner,
		uuidGenerator:             generator,
		provisioningQueue:         provisioningQueue,
		deprovisioningQueue:       deprovisioningQueue,
		shootUpgradeQueue:         shootUpgradeQueue,
		shootProvider:             shootProvider,
		dynamicKubeconfigProvider: dynamicKubeconfigProvider,
	}
}

func (r *service) ProvisionRuntime(config gqlschema.ProvisionRuntimeInput, tenant, subAccount string) (*gqlschema.OperationStatus, apperrors.AppError) {
	runtimeInput := config.RuntimeInput

	var runtimeID string

	err := util.RetryOnError(5*time.Second, 3, "Error while registering runtime in Director: %s", func() (err apperrors.AppError) {
		runtimeID, err = r.directorService.CreateRuntime(runtimeInput, tenant)
		return
	})

	if err != nil {
		return nil, err.Append("Failed to register Runtime")
	}

	cluster, err := r.inputConverter.ProvisioningInputToCluster(runtimeID, config, tenant, subAccount)
	if err != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, err
	}

	dbSession, dberr := r.dbSessionFactory.NewSessionWithinTransaction()
	if dberr != nil {
		return nil, dberr
	}
	defer dbSession.RollbackUnlessCommitted()

	// Try to set provisioning started before triggering it (which is hard to interrupt) to verify all unique constraints
	operation, dberr := r.setProvisioningStarted(dbSession, runtimeID, cluster)
	if dberr != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, dberr
	}

	err = r.provisioner.ProvisionCluster(cluster, operation.ID)
	if err != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, err.Append("Failed to start provisioning")
	}

	dberr = dbSession.Commit()
	if dberr != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, dberr
	}

	log.Infof("KymaConfig not provided. Starting provisioning steps for runtime %s without installation", cluster.ID)
	r.provisioningQueue.Add(operation.ID)

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) unregisterFailedRuntime(id, tenant string) {
	log.Infof("Starting provisioning failed. Unregistering Runtime %s...", id)
	err := util.RetryOnError(10*time.Second, 3, "Error while unregistering runtime in Director: %s", func() (err apperrors.AppError) {
		err = r.directorService.DeleteRuntime(id, tenant)
		return
	})
	if err != nil {
		log.Warnf("Failed to unregister failed Runtime '%s': %s", id, err.Error())
	}
}

func (r *service) DeprovisionRuntime(id string) (string, apperrors.AppError) {
	session := r.dbSessionFactory.NewReadWriteSession()

	appErr := r.verifyLastOperationFinished(session, id)
	if appErr != nil {
		return "", appErr
	}

	cluster, dberr := session.GetCluster(id)
	if dberr != nil {
		return "", dberr
	}

	operation, appErr := r.provisioner.DeprovisionCluster(cluster, r.uuidGenerator.New())
	if appErr != nil {
		return "", apperrors.Internal("Failed to start deprovisioning: %s", appErr.Error()).SetComponent(appErr.Component()).SetReason(appErr.Reason())
	}

	dberr = session.InsertOperation(operation)
	if dberr != nil {
		return "", dberr
	}

	log.Infof("Starting deprovisioning steps for runtime %s without installation", cluster.ID)
	r.deprovisioningQueue.Add(operation.ID)

	return operation.ID, nil
}

func (r *service) UpgradeGardenerShoot(runtimeID string, input gqlschema.UpgradeShootInput) (*gqlschema.OperationStatus, apperrors.AppError) {
	log.Infof("Starting Upgrade of Gardener Shoot for Runtime '%s'...", runtimeID)

	if input.GardenerConfig == nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Error: Gardener config is nil")
	}

	session := r.dbSessionFactory.NewReadSession()

	err := r.verifyLastOperationFinished(session, runtimeID)
	if err != nil {
		return &gqlschema.OperationStatus{}, err
	}

	cluster, dberr := session.GetCluster(runtimeID)
	if dberr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Failed to find shoot cluster to upgrade in database: %s", dberr.Error())
	}

	gardenerConfig, err := r.inputConverter.UpgradeShootInputToGardenerConfig(*input.GardenerConfig, cluster.ClusterConfig)
	if err != nil {
		return &gqlschema.OperationStatus{}, err.Append("Failed to convert GardenerClusterUpgradeConfig: %s", err.Error())
	}

	shoot, err := r.shootProvider.Get(runtimeID, cluster.Tenant)
	if err != nil {
		return &gqlschema.OperationStatus{}, err.Append("Failed to get shoot")
	}

	// This is a workaround for a problem with Kubernetes auto upgrade. If Kubernetes gets updated the current Kubernetes version is obtained for the shoot and stored in the database.
	shouldTakeShootKubernetesVersion, err := isVersionHigher(shoot.Spec.Kubernetes.Version, gardenerConfig.KubernetesVersion)
	if err != nil {
		return &gqlschema.OperationStatus{}, err.Append("Failed to check if the shoot kubernetes version is higher than the config one")
	}
	if shouldTakeShootKubernetesVersion {
		log.Infof("Kubernetes version in shoot was higher than the version provided in UpgradeGardenerShoot. Version fetched from the shoot will be used :%s.", shoot.Spec.Kubernetes.Version)
		gardenerConfig.KubernetesVersion = shoot.Spec.Kubernetes.Version
	}

	// This is a workaround for the possible manual modification of the Shoot Spec Extensions. If ShootNetworkingFilterDisabled is modified manually, Provisioner should use the actual value.
	shootNetworkingFilterDisabled := getShootNetworkingFilterDisabled(shoot.Spec.Extensions)
	if input.GardenerConfig.ShootNetworkingFilterDisabled == nil && shootNetworkingFilterDisabled != nil {
		log.Warnf("ShootNetworkingFilter extension was different than the one provided in UpgradeGardenerShoot. Value fetched from the shoot will be used: %t.", *shootNetworkingFilterDisabled)
		gardenerConfig.ShootNetworkingFilterDisabled = shootNetworkingFilterDisabled
	}

	// Validate provider specific changes to the shoot
	err = gardenerConfig.GardenerProviderConfig.ValidateShootConfigChange(&shoot)
	if err != nil {
		return &gqlschema.OperationStatus{}, err.Append("Invalid gardener provider config change")
	}
	txSession, dbErr := r.dbSessionFactory.NewSessionWithinTransaction()
	if dbErr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Failed to start database transaction: %s", dbErr.Error())
	}
	defer txSession.RollbackUnlessCommitted()

	operation, gardError := r.setGardenerShootUpgradeStarted(txSession, cluster, gardenerConfig, input.Administrators)
	if gardError != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Failed to set shoot upgrade started: %s", gardError.Error())
	}

	err = r.provisioner.UpgradeCluster(cluster.ID, gardenerConfig)
	if err != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Failed to upgrade Cluster: %s", err.Error())
	}

	dbErr = txSession.Commit()
	if dbErr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("Failed to commit upgrade transaction: %s", dbErr.Error())
	}

	r.shootUpgradeQueue.Add(operation.ID)

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) verifyLastOperationFinished(session dbsession.ReadSession, runtimeId string) apperrors.AppError {
	lastOperation, dberr := session.GetLastOperation(runtimeId)
	if dberr != nil {
		return dberr.Append("failed to get last operation")
	}

	if lastOperation.State == model.InProgress {
		return apperrors.BadRequest("cannot start new operation for %s Runtime while previous one is in progress", runtimeId)
	}

	return nil
}

func (r *service) ReconnectRuntimeAgent(string) (string, apperrors.AppError) {
	return "", nil
}

func (r *service) RuntimeStatus(runtimeID string) (*gqlschema.RuntimeStatus, apperrors.AppError) {
	runtimeStatus, dberr := r.getRuntimeStatus(runtimeID)
	if dberr != nil {
		return nil, dberr.Append("failed to get Runtime Status")
	}

	return r.graphQLConverter.RuntimeStatusToGraphQLStatus(runtimeStatus), nil
}

func (r *service) RuntimeOperationStatus(operationID string) (*gqlschema.OperationStatus, apperrors.AppError) {
	readSession := r.dbSessionFactory.NewReadSession()

	operation, dberr := readSession.GetOperation(operationID)
	if dberr != nil {
		return nil, dberr.Append("failed to get Runtime Operation Status")
	}

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) getRuntimeStatus(runtimeID string) (model.RuntimeStatus, apperrors.AppError) {
	session := r.dbSessionFactory.NewReadSession()

	operation, err := session.GetLastOperation(runtimeID)
	if err != nil {
		return model.RuntimeStatus{}, err
	}

	cluster, err := session.GetCluster(runtimeID)
	if err != nil {
		return model.RuntimeStatus{}, err
	}

	kubeconfig, fetchErr := r.dynamicKubeconfigProvider.FetchFromRequest(cluster.ClusterConfig.Name)
	if fetchErr != nil {
		return model.RuntimeStatus{}, apperrors.Internal("unable to fetch kubeconfig: %s", fetchErr)
	}

	cluster.Kubeconfig = util.PtrTo(string(kubeconfig))

	return model.RuntimeStatus{
		LastOperationStatus:  operation,
		RuntimeConfiguration: cluster,
	}, nil
}

func (r *service) setProvisioningStarted(dbSession dbsession.WriteSession, runtimeID string, cluster model.Cluster) (model.Operation, dberrors.Error) {
	timestamp := time.Now()
	cluster.CreationTimestamp = timestamp

	if err := dbSession.InsertCluster(cluster); err != nil {
		return model.Operation{}, dberrors.Internal("Failed to set provisioning started: %s", err)
	}

	if err := dbSession.InsertGardenerConfig(cluster.ClusterConfig); err != nil {
		return model.Operation{}, dberrors.Internal("Failed to set provisioning started: %s", err)
	}

	provisioningMode := model.Provision

	operation, err := r.setOperationStarted(dbSession, runtimeID, provisioningMode, model.WaitingForClusterDomain, timestamp, "Provisioning started")
	if err != nil {
		return model.Operation{}, err.Append("Failed to set provisioning started: %s")
	}

	return operation, nil
}

func (r *service) setGardenerShootUpgradeStarted(txSession dbsession.WriteSession, currentCluster model.Cluster, gardenerConfig model.GardenerConfig, administrators []string) (model.Operation, error) {
	log.Infof("Starting Upgrade of Gardener Shoot operation")

	dberr := txSession.UpdateGardenerClusterConfig(gardenerConfig)
	if dberr != nil {
		return model.Operation{}, dberrors.Internal("Failed to set Shoot Upgrade started: %s", dberr.Error())
	}

	dberr = txSession.InsertAdministrators(currentCluster.ID, administrators)
	if dberr != nil {
		return model.Operation{}, dberrors.Internal("Failed to set Shoot Upgrade started: %s", dberr.Error())
	}

	operation, dbError := r.setOperationStarted(txSession, currentCluster.ID, model.UpgradeShoot, model.WaitingForShootNewVersion, time.Now(), "Starting Gardener Shoot upgrade")

	if dbError != nil {
		return model.Operation{}, dbError.Append("Failed to start operation of Gardener Shoot upgrade %s", dbError.Error())
	}

	return operation, nil
}

func (r *service) setOperationStarted(
	dbSession dbsession.WriteSession,
	runtimeID string,
	operationType model.OperationType,
	operationStage model.OperationStage,
	timestamp time.Time,
	message string) (model.Operation, dberrors.Error) {
	id := r.uuidGenerator.New()

	operation := model.Operation{
		ID:             id,
		Type:           operationType,
		StartTimestamp: timestamp,
		State:          model.InProgress,
		Message:        message,
		ClusterID:      runtimeID,
		Stage:          operationStage,
		LastTransition: &timestamp,
	}

	err := dbSession.InsertOperation(operation)
	if err != nil {
		return model.Operation{}, err.Append("failed to insert operation")
	}

	return operation, nil
}

func isVersionHigher(version1, version2 string) (bool, apperrors.AppError) {
	parsedVersion1, err := version.NewVersion(version1)
	if err != nil {
		return false, apperrors.Internal("Failed to parse \"%s\" as a version", version1)
	}
	parsedVersion2, err := version.NewVersion(version2)
	if err != nil {
		return false, apperrors.Internal("Failed to parse \"%s\" as a version", version2)
	}
	return parsedVersion1.GreaterThan(parsedVersion2), nil
}

func getShootNetworkingFilterDisabled(extensions []gardener_Types.Extension) *bool {
	for _, extension := range extensions {
		if extension.Type == model.ShootNetworkingFilterExtensionType {
			return extension.Disabled
		}
	}
	return nil
}
