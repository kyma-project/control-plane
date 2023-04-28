package deprovisioning

import (
	"context"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CleanupClusterStep struct {
	installationService installation.Service
	nextStep            model.OperationStage
	timeLimit           time.Duration
	gardenerClient      GardenerClient
}

func NewCleanupClusterStep(gardenerClient GardenerClient, installationService installation.Service, nextStep model.OperationStage, timeLimit time.Duration) *CleanupClusterStep {
	return &CleanupClusterStep{
		installationService: installationService,
		nextStep:            nextStep,
		timeLimit:           timeLimit,
		gardenerClient:      gardenerClient,
	}
}

func (s *CleanupClusterStep) Name() model.OperationStage {
	return model.CleanupCluster
}

func (s *CleanupClusterStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *CleanupClusterStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {
	logger.Debugf("Starting cleanup cluster step for %s ...", cluster.ID)
	if cluster.Kubeconfig == nil {
		// Kubeconfig can be nil if Gardener failed to create cluster. We must go to the next step to finalize deprovisioning
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	shoot, err := s.gardenerClient.Get(context.Background(), cluster.ClusterConfig.Name, metav1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
	}

	if shoot.Status.IsHibernated {
		// The cluster is hibernated we must go to the next step
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}
