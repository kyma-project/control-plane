package shootupgrade

import (
	"context"
	"errors"
	"fmt"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GardenerClient interface {
	Get(ctx context.Context, name string, options v1.GetOptions) (*gardener_types.Shoot, error)
}

type WaitForShootUpgradeStep struct {
	gardenerClient GardenerClient
	nextStep       model.OperationStage
	timeLimit      time.Duration
}

func NewWaitForShootUpgradeStep(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForShootUpgradeStep {
	return &WaitForShootUpgradeStep{
		gardenerClient: gardenerClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (s WaitForShootUpgradeStep) Name() model.OperationStage {
	return model.WaitingForShootUpgrade
}

func (s *WaitForShootUpgradeStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForShootUpgradeStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

	gardenerConfig := cluster.ClusterConfig

	shoot, err := s.gardenerClient.Get(context.Background(), gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, err
	}

	lastOperation := shoot.Status.LastOperation

	if lastOperation != nil {
		if lastOperation.State == gardencorev1beta1.LastOperationStateSucceeded {
			return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
		}

		if lastOperation.State == gardencorev1beta1.LastOperationStateFailed {
			if gardencorev1beta1helper.HasErrorCode(shoot.Status.LastErrors, gardencorev1beta1.ErrorInfraRateLimitsExceeded) {
				return operations.StageResult{}, errors.New("error during shoot cluster upgrade: rate limits exceeded")
			}
			logger.Warningf("Gardener Shoot cluster upgrade operation failed! Last state: %s, Description: %s", lastOperation.State, lastOperation.Description)

			err := errors.New(fmt.Sprintf("Gardener Shoot cluster upgrade failed. Last Shoot state: %s, Shoot description: %s", lastOperation.State, lastOperation.Description))

			return operations.StageResult{}, operations.NewNonRecoverableError(err)
		}
	}

	return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
}
