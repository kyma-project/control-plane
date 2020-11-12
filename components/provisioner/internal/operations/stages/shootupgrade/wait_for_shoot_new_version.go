package shootupgrade

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WaitForShootNewVersionStep struct {
	gardenerClient GardenerClient
	nextStep       model.OperationStage
	timeLimit      time.Duration
}

func NewWaitForShootNewVersionStep(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForShootNewVersionStep {
	return &WaitForShootNewVersionStep{
		gardenerClient: gardenerClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (s WaitForShootNewVersionStep) Name() model.OperationStage {
	return model.WaitingForShootNewVersion
}

func (s *WaitForShootNewVersionStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForShootNewVersionStep) Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

	gardenerConfig := cluster.ClusterConfig

	shoot, err := s.gardenerClient.Get(context.Background(), gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, err
	}

	if shoot.Status.ObservedGeneration == shoot.ObjectMeta.Generation {
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	if shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateFailed {
		err := fmt.Errorf(fmt.Sprintf("Gardener Shoot cluster upgrade failed. Last Shoot state: %s, Shoot description: %s", shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description))
		return operations.StageResult{}, operations.NewNonRecoverableError(err)
	}

	return operations.StageResult{Stage: s.Name(), Delay: 5 * time.Second}, nil
}
