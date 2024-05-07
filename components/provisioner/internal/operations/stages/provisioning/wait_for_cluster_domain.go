package provisioning

import (
	"context"
	"time"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
)

type WaitForClusterDomainStep struct {
	gardenerClient GardenerClient
	nextStep       model.OperationStage
	timeLimit      time.Duration
}

//go:generate mockery --name=GardenerClient
type GardenerClient interface {
	Get(ctx context.Context, name string, options v1.GetOptions) (*gardener_types.Shoot, error)
}

func NewWaitForClusterDomainStep(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForClusterDomainStep {
	return &WaitForClusterDomainStep{
		gardenerClient: gardenerClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (s *WaitForClusterDomainStep) Name() model.OperationStage {
	return model.WaitingForClusterDomain
}

func (s *WaitForClusterDomainStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForClusterDomainStep) Run(cluster model.Cluster, _ model.Operation, log logrus.FieldLogger) (operations.StageResult, error) {
	shoot, err := s.gardenerClient.Get(context.Background(), cluster.ClusterConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, util.K8SErrorToAppError(err).SetComponent(apperrors.ErrGardenerClient)
	}

	if shoot.Spec.DNS == nil || shoot.Spec.DNS.Domain == nil {
		log.Warnf("DNS Domain is not set yet for runtime ID: %s", cluster.ID)
		return operations.StageResult{Stage: s.Name(), Delay: 5 * time.Second}, nil
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}
