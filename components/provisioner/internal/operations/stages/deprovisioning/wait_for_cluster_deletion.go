package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WaitForClusterDeletionStep struct {
	gardenerClient GardenerClient
	dbsFactory     dbsession.Factory
	directorClient director.DirectorClient
	nextStep       model.OperationStage
	timeLimit      time.Duration
}

func NewWaitForClusterDeletionStep(gardenerClient GardenerClient, dbsFactory dbsession.Factory, directorClient director.DirectorClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForClusterDeletionStep {
	return &WaitForClusterDeletionStep{
		gardenerClient: gardenerClient,
		dbsFactory:     dbsFactory,
		directorClient: directorClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (s *WaitForClusterDeletionStep) Name() model.OperationStage {
	return model.WaitForClusterDeletion
}

func (s *WaitForClusterDeletionStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForClusterDeletionStep) Run(cluster model.Cluster, _ model.Operation, _ logrus.FieldLogger) (operations.StageResult, error) {

	shootExists, err := s.shootExists(cluster.ClusterConfig.Name)
	if err != nil {
		return operations.StageResult{}, err
	}

	if shootExists {
		return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
	}

	err = s.setDeprovisioningFinished(cluster)
	if err != nil {
		return operations.StageResult{}, err
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}

func (s *WaitForClusterDeletionStep) shootExists(gardenerClusterName string) (bool, error) {
	_, err := s.gardenerClient.Get(context.Background(), gardenerClusterName, v1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
	}

	return true, nil
}

func (s *WaitForClusterDeletionStep) setDeprovisioningFinished(cluster model.Cluster) error {
	session, dberr := s.dbsFactory.NewSessionWithinTransaction()
	if dberr != nil {
		return errors.Wrap(dberr, "error starting db session with transaction")
	}
	defer session.RollbackUnlessCommitted()

	dberr = session.MarkClusterAsDeleted(cluster.ID)
	if dberr != nil {
		return errors.Wrap(dberr, "error marking cluster for deletion")
	}

	err := s.deleteRuntime(cluster)
	if err != nil {
		return err
	}

	dberr = session.Commit()
	if dberr != nil {
		return errors.Wrap(dberr, "error commiting transaction")
	}

	return nil
}

func (s *WaitForClusterDeletionStep) deleteRuntime(cluster model.Cluster) error {
	var exists bool
	err := util.RetryOnError(5*time.Second, 3, "Error while checking if runtime exists in Director: %s", func() (err apperrors.AppError) {
		exists, err = s.directorClient.RuntimeExists(cluster.ID, cluster.Tenant)
		return
	})

	if err != nil {
		return errors.Wrap(err, "error checking Runtime exists in Director")
	}

	if !exists {
		return nil
	}

	err = util.RetryOnError(5*time.Second, 3, "Error while unregistering runtime in Director: %s", func() (err apperrors.AppError) {
		err = s.directorClient.DeleteRuntime(cluster.ID, cluster.Tenant)
		return
	})

	if err != nil {
		return errors.Wrap(err, "error deleting Runtime form Director")
	}

	return nil
}
