package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"

	uuid "github.com/kyma-project/control-plane/components/provisioner/internal/uuid"

	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/dberrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"

	"github.com/kyma-project/control-plane/components/provisioner/internal/director"

	log "github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

//go:generate mockery -name=Service
type Service interface {
	ProvisionRuntime(config gqlschema.ProvisionRuntimeInput, tenant, subAccount string) (*gqlschema.OperationStatus, apperrors.AppError)
	UpgradeRuntime(id string, config gqlschema.UpgradeRuntimeInput) (*gqlschema.OperationStatus, apperrors.AppError)
	DeprovisionRuntime(id string) (string, apperrors.AppError)
	UpgradeGardenerShoot(id string, input gqlschema.UpgradeShootInput) (*gqlschema.OperationStatus, apperrors.AppError)
	ReconnectRuntimeAgent(id string) (string, apperrors.AppError)
	RuntimeStatus(id string) (*gqlschema.RuntimeStatus, apperrors.AppError)
	RuntimeOperationStatus(id string) (*gqlschema.OperationStatus, apperrors.AppError)
	RollBackLastUpgrade(runtimeID string) (*gqlschema.RuntimeStatus, apperrors.AppError)
	HibernateCluster(clusterID string) (*gqlschema.OperationStatus, apperrors.AppError)
}

//go:generate mockery -name=Provisioner
type Provisioner interface {
	ProvisionCluster(cluster model.Cluster, operationId string) apperrors.AppError
	DeprovisionCluster(cluster model.Cluster, operationId string) (model.Operation, apperrors.AppError)
	UpgradeCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError
	HibernateCluster(clusterID string, upgradeConfig model.GardenerConfig) apperrors.AppError
	GetHibernationStatus(clusterID string, gardenerConfig model.GardenerConfig) (model.HibernationStatus, apperrors.AppError)
}

//go:generate mockery -name=KubernetesVersionProvider
type KubernetesVersionProvider interface {
	Get(runtimeID string, tenant string) (string, apperrors.AppError)
}

type service struct {
	inputConverter            InputConverter
	graphQLConverter          GraphQLConverter
	directorService           director.DirectorClient
	kubernetesVersionProvider KubernetesVersionProvider
	installationClient        installation.Service

	dbSessionFactory dbsession.Factory
	provisioner      Provisioner
	uuidGenerator    uuid.UUIDGenerator

	provisioningQueue            queue.OperationQueue
	provisioningNoInstallQueue   queue.OperationQueue
	deprovisioningQueue          queue.OperationQueue
	deprovisioningNoInstallQueue queue.OperationQueue
	upgradeQueue                 queue.OperationQueue
	shootUpgradeQueue            queue.OperationQueue
	hibernationQueue             queue.OperationQueue
}

func NewProvisioningService(
	inputConverter InputConverter,
	graphQLConverter GraphQLConverter,
	directorService director.DirectorClient,
	factory dbsession.Factory,
	provisioner Provisioner,
	generator uuid.UUIDGenerator,
	kubernetesVersionProvider KubernetesVersionProvider,
	installationClient installation.Service,
	provisioningQueue queue.OperationQueue,
	provisioningNoInstallQueue queue.OperationQueue,
	deprovisioningQueue queue.OperationQueue,
	deprovisioningNoInstallQueue queue.OperationQueue,
	upgradeQueue queue.OperationQueue,
	shootUpgradeQueue queue.OperationQueue,
	hibernationQueue queue.OperationQueue,

) Service {
	return &service{
		inputConverter:               inputConverter,
		graphQLConverter:             graphQLConverter,
		directorService:              directorService,
		dbSessionFactory:             factory,
		provisioner:                  provisioner,
		uuidGenerator:                generator,
		provisioningQueue:            provisioningQueue,
		provisioningNoInstallQueue:   provisioningNoInstallQueue,
		deprovisioningQueue:          deprovisioningQueue,
		deprovisioningNoInstallQueue: deprovisioningNoInstallQueue,
		upgradeQueue:                 upgradeQueue,
		shootUpgradeQueue:            shootUpgradeQueue,
		hibernationQueue:             hibernationQueue,
		kubernetesVersionProvider:    kubernetesVersionProvider,
		installationClient:           installationClient,
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
		return nil, apperrors.Internal("Failed to start database transaction: %s", dberr.Error())
	}
	defer dbSession.RollbackUnlessCommitted()

	withKymaConfig := config.KymaConfig != nil

	// Try to set provisioning started before triggering it (which is hard to interrupt) to verify all unique constraints
	operation, dberr := r.setProvisioningStarted(dbSession, runtimeID, cluster, withKymaConfig)
	if dberr != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, apperrors.Internal(dberr.Error())
	}

	err = r.provisioner.ProvisionCluster(cluster, operation.ID)
	if err != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, err.Append("Failed to start provisioning")
	}

	dberr = dbSession.Commit()
	if dberr != nil {
		r.unregisterFailedRuntime(runtimeID, tenant)
		return nil, apperrors.Internal("Failed to commit transaction: %s", dberr.Error())
	}

	if withKymaConfig {
		log.Infof("KymaConfig provided. Starting provisioning steps for runtime %s with installation", cluster.ID)
		r.provisioningQueue.Add(operation.ID)
	} else {
		log.Infof("KymaConfig not provided. Starting provisioning steps for runtime %s without installation", cluster.ID)
		r.provisioningNoInstallQueue.Add(operation.ID)
	}

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
		return "", apperrors.Internal("Failed to get cluster: %s", dberr.Error())
	}

	operation, appErr := r.provisioner.DeprovisionCluster(cluster, r.uuidGenerator.New())
	if appErr != nil {
		return "", apperrors.Internal("Failed to start deprovisioning: %s", appErr.Error())
	}

	dberr = session.InsertOperation(operation)
	if dberr != nil {
		return "", apperrors.Internal("Failed to insert operation to database: %s", dberr.Error())
	}

	if cluster.Kubeconfig == nil {
		return "", apperrors.Internal("error: kubeconfig is nil")
	}

	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		return "", apperrors.Internal("error: failed to create kubernetes config from raw: %s", err.Error())
	}

	installationState, err := r.installationClient.CheckInstallationState(k8sConfig)

	// TODO remove this log message!!!
	// This is only only for debug
	if err != nil {
		log.Infof("Error while checking installation state :%s.", err.Error())
	}

	log.Infof("Installation state is :%s.", installationState.State)

	if err == nil && util.NotNilOrEmpty(cluster.ActiveKymaConfigId) && installationState.State == string(v1alpha1.StateInstalled) {
		// "Kyma classic" mode
		log.Infof("Starting deprovisioning steps for runtime %s with installation", cluster.ID)
		r.deprovisioningQueue.Add(operation.ID)
	} else {
		// Kyma 2.0 mode
		log.Infof("Starting deprovisioning steps for runtime %s without installation", cluster.ID)
		r.deprovisioningNoInstallQueue.Add(operation.ID)
	}

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

	// We need to fetch current Kubernetes version to be able to maintain consistency
	shootKubernetesVersion, err := r.kubernetesVersionProvider.Get(runtimeID, cluster.Tenant)
	if err != nil {
		return nil, err
	}

	// This is a workaround for a problem with Kubernetes auto upgrade. If Kubernetes gets updated the current Kubernetes version is obtained for the shoot and stored in the database.
	if shootKubernetesVersion > gardenerConfig.KubernetesVersion {
		log.Infof("Kubernetes version in shoot was higher than the version provided in UpgradeGardenerShoot. Version fetched from the shoot will be used :%s.", shootKubernetesVersion)
		gardenerConfig.KubernetesVersion = shootKubernetesVersion
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

func (r *service) HibernateCluster(runtimeID string) (*gqlschema.OperationStatus, apperrors.AppError) {
	log.Infof("Starting hibernation for Runtime '%s'...", runtimeID)

	session := r.dbSessionFactory.NewReadSession()

	err := r.verifyLastOperationFinished(session, runtimeID)
	if err != nil {
		return nil, err
	}

	cluster, dberr := session.GetCluster(runtimeID)
	if dberr != nil {
		return nil, apperrors.Internal("Failed to find shoot cluster to hibernate in database: %s", dberr.Error())
	}

	txSession, dbErr := r.dbSessionFactory.NewSessionWithinTransaction()
	if dbErr != nil {
		return nil, apperrors.Internal("Failed to start database transaction: %s", dbErr.Error())
	}
	defer txSession.RollbackUnlessCommitted()

	operation, gardError := r.setHibernationStarted(txSession, cluster, cluster.ClusterConfig)
	if gardError != nil {
		return nil, apperrors.Internal("Failed to set hibernation started: %s", gardError.Error())
	}

	err = r.provisioner.HibernateCluster(cluster.ID, cluster.ClusterConfig)
	if err != nil {
		return nil, apperrors.Internal("Failed to hibernate Cluster: %s", err.Error())
	}

	dbErr = txSession.Commit()
	if dbErr != nil {
		return nil, apperrors.Internal("Failed to commit hibernation transaction: %s", dbErr.Error())
	}

	r.hibernationQueue.Add(operation.ID)

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) verifyLastOperationFinished(session dbsession.ReadSession, runtimeId string) apperrors.AppError {
	lastOperation, dberr := session.GetLastOperation(runtimeId)
	if dberr != nil {
		return apperrors.Internal("failed to get last operation: %s", dberr.Error())
	}

	if lastOperation.State == model.InProgress {
		return apperrors.BadRequest("cannot start new operation for %s Runtime while previous one is in progress", runtimeId)
	}

	return nil
}

func (r *service) UpgradeRuntime(runtimeId string, input gqlschema.UpgradeRuntimeInput) (*gqlschema.OperationStatus, apperrors.AppError) {
	if input.KymaConfig == nil {
		return &gqlschema.OperationStatus{}, apperrors.BadRequest("error: Kyma config is nil")
	}

	session := r.dbSessionFactory.NewReadSession()

	if err := r.verifyLastOperationFinished(session, runtimeId); err != nil {
		return &gqlschema.OperationStatus{}, err
	}

	kymaConfig, err := r.inputConverter.KymaConfigFromInput(runtimeId, *input.KymaConfig)
	if err != nil {
		return &gqlschema.OperationStatus{}, err.Append("failed to convert KymaConfigInput")
	}

	cluster, dberr := session.GetCluster(runtimeId)
	if dberr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("failed to read cluster from database: %s", dberr.Error())
	}

	if util.IsNilOrEmpty(cluster.ActiveKymaConfigId) {
		return &gqlschema.OperationStatus{}, apperrors.Internal("failed to upgrade cluster: %s Kyma configuration of the cluster is managed by Reconciler", cluster.ID)
	}

	// We need to fetch current Kubernetes version to be able to maintain consistency
	shootKubernetesVersion, err := r.kubernetesVersionProvider.Get(runtimeId, cluster.Tenant)
	if err != nil {
		return nil, err
	}

	txSession, dberr := r.dbSessionFactory.NewSessionWithinTransaction()
	if dberr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("failed to start database transaction: %s", dberr.Error())
	}
	defer txSession.RollbackUnlessCommitted()

	operation, dberr := r.setUpgradeStarted(txSession, cluster, kymaConfig)
	if dberr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("failed to set upgrade started: %s", dberr.Error())
	}

	// This is a workaround for a problem with Kubernetes auto upgrade. If Kubernetes gets updated the current Kubernetes version is obtained for the shoot and stored in the database.

	if shootKubernetesVersion > cluster.ClusterConfig.KubernetesVersion {
		log.Infof("Kubernetes version in shoot was higher than the version stored in database. Version fetched from the shoot will be stored in database :%s.", shootKubernetesVersion)
		dberr = txSession.UpdateKubernetesVersion(runtimeId, shootKubernetesVersion)
		if dberr != nil {
			return &gqlschema.OperationStatus{}, apperrors.Internal("failed to set Kubernetes version: %s", dberr.Error())
		}
	}

	dberr = txSession.Commit()
	if dberr != nil {
		return &gqlschema.OperationStatus{}, apperrors.Internal("failed to commit upgrade transaction: %s", dberr.Error())
	}

	r.upgradeQueue.Add(operation.ID)

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) ReconnectRuntimeAgent(id string) (string, apperrors.AppError) {
	return "", nil
}

func (r *service) RuntimeStatus(runtimeID string) (*gqlschema.RuntimeStatus, apperrors.AppError) {
	runtimeStatus, dberr := r.getRuntimeStatus(runtimeID)
	if dberr != nil {
		return nil, apperrors.Internal("failed to get Runtime Status: %s", dberr.Error())
	}

	return r.graphQLConverter.RuntimeStatusToGraphQLStatus(runtimeStatus), nil
}

func (r *service) RuntimeOperationStatus(operationID string) (*gqlschema.OperationStatus, apperrors.AppError) {
	readSession := r.dbSessionFactory.NewReadSession()

	operation, dberr := readSession.GetOperation(operationID)
	if dberr != nil {
		return nil, apperrors.Internal("failed to get Runtime Operation Status: %s", dberr.Error())
	}

	return r.graphQLConverter.OperationStatusToGQLOperationStatus(operation), nil
}

func (r *service) RollBackLastUpgrade(runtimeID string) (*gqlschema.RuntimeStatus, apperrors.AppError) {

	readSession := r.dbSessionFactory.NewReadSession()

	lastOp, err := readSession.GetLastOperation(runtimeID)
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}

	if lastOp.Type != model.Upgrade || lastOp.State == model.InProgress {
		return nil, apperrors.BadRequest("error: upgrade can be rolled back only if it is the last operation that is already finished")
	}

	runtimeUpgrade, err := readSession.GetRuntimeUpgrade(lastOp.ID)
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}

	txSession, err := r.dbSessionFactory.NewSessionWithinTransaction()
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}
	defer txSession.RollbackUnlessCommitted()

	err = txSession.SetActiveKymaConfig(runtimeID, runtimeUpgrade.PreUpgradeKymaConfigId)
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}

	err = txSession.UpdateUpgradeState(lastOp.ID, model.UpgradeRolledBack)
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}

	err = txSession.Commit()
	if err != nil {
		return nil, apperrors.Internal("error rolling back last upgrade: %s", err.Error())
	}

	return r.RuntimeStatus(runtimeID)
}

func (r *service) getRuntimeStatus(runtimeID string) (model.RuntimeStatus, error) {
	session := r.dbSessionFactory.NewReadSession()

	operation, err := session.GetLastOperation(runtimeID)
	if err != nil {
		return model.RuntimeStatus{}, err
	}

	cluster, err := session.GetCluster(runtimeID)
	if err != nil {
		return model.RuntimeStatus{}, err
	}

	hibernationStatus, apperr := r.provisioner.GetHibernationStatus(runtimeID, cluster.ClusterConfig)
	if apperr != nil {
		return model.RuntimeStatus{}, apperr
	}

	return model.RuntimeStatus{
		LastOperationStatus:  operation,
		RuntimeConfiguration: cluster,
		HibernationStatus:    hibernationStatus,
	}, nil
}

func (r *service) setProvisioningStarted(dbSession dbsession.WriteSession, runtimeID string, cluster model.Cluster, withKymaConfig bool) (model.Operation, dberrors.Error) {
	timestamp := time.Now()
	cluster.CreationTimestamp = timestamp

	if err := dbSession.InsertCluster(cluster); err != nil {
		return model.Operation{}, dberrors.Internal("Failed to set provisioning started: %s", err)
	}

	if err := dbSession.InsertGardenerConfig(cluster.ClusterConfig); err != nil {
		return model.Operation{}, dberrors.Internal("Failed to set provisioning started: %s", err)
	}

	provisioningMode := model.ProvisionNoInstall
	if withKymaConfig {
		provisioningMode = model.Provision
		if err := dbSession.InsertKymaConfig(*cluster.KymaConfig); err != nil {
			return model.Operation{}, dberrors.Internal("Failed to set provisioning started: %s", err)
		}
	}

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

func (r *service) setUpgradeStarted(txSession dbsession.WriteSession, cluster model.Cluster, kymaConfig model.KymaConfig) (model.Operation, dberrors.Error) {

	err := txSession.InsertKymaConfig(kymaConfig)
	if err != nil {
		return model.Operation{}, err.Append("Failed to insert Kyma Config")
	}

	operation, err := r.setOperationStarted(txSession, cluster.ID, model.Upgrade, model.StartingUpgrade, time.Now(), "Starting Kyma upgrade")
	if err != nil {
		return model.Operation{}, err.Append("Failed to set operation started")
	}

	runtimeUpgrade := model.RuntimeUpgrade{
		Id:                      r.uuidGenerator.New(),
		State:                   model.UpgradeInProgress,
		OperationId:             operation.ID,
		PreUpgradeKymaConfigId:  cluster.KymaConfig.ID,
		PostUpgradeKymaConfigId: kymaConfig.ID,
	}

	err = txSession.InsertRuntimeUpgrade(runtimeUpgrade)
	if err != nil {
		return model.Operation{}, err.Append("Failed to insert Runtime Upgrade")
	}

	err = txSession.SetActiveKymaConfig(cluster.ID, kymaConfig.ID)
	if err != nil {
		return model.Operation{}, err.Append("Failed to update Kyma config in cluster")
	}

	return operation, nil
}

func (r *service) setHibernationStarted(txSession dbsession.WriteSession, currentCluster model.Cluster, gardenerConfig model.GardenerConfig) (model.Operation, error) {
	log.Infof("Starting hibernation operation")

	operation, dbError := r.setOperationStarted(txSession, currentCluster.ID, model.Hibernate, model.WaitForHibernation, time.Now(), "Starting ")

	if dbError != nil {
		return model.Operation{}, dbError.Append("Failed to start hibernation operation:  %s", dbError.Error())
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
