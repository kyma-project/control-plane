package shootupgrade

import (
	"errors"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WaitForShootClusterNewVersionStep struct {
	gardenerClient         GardenerClient
	nextStep               model.OperationStage
	timeLimit              time.Duration
	initialResourceVersion string
}

func NewWaitForShootNewVersionStep(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForShootClusterNewVersionStep {
	return &WaitForShootClusterNewVersionStep{
		gardenerClient: gardenerClient,
		nextStep:       nextStep,
		timeLimit:      timeLimit,
	}
}

func (s WaitForShootClusterNewVersionStep) Name() model.OperationStage {
	return model.WaitingForShootNewVersion
}

func (s *WaitForShootClusterNewVersionStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForShootClusterNewVersionStep) Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

	gardenerConfig := cluster.ClusterConfig

	shoot, err := s.gardenerClient.Get(gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, err
	}

	if shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateFailed {
		err := errors.New(fmt.Sprintf("Gardener Shoot cluster upgrade failed. Last Shoot state: %s, Shoot description: %s", shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description))
		return operations.StageResult{}, operations.NewNonRecoverableError(err)
	}

	if s.initialResourceVersion == "" {
		s.initialResourceVersion = shoot.ObjectMeta.ResourceVersion
		logger.Info("Initial resource version: ", s.initialResourceVersion)
		return operations.StageResult{Stage: s.Name(), Delay: 5 * time.Second}, nil
	}

	if s.initialResourceVersion != shoot.ObjectMeta.ResourceVersion {
		logger.Info("Shoot upgrade operation has generated new resource version: ", shoot.ObjectMeta.ResourceVersion)
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	return operations.StageResult{Stage: s.Name(), Delay: 5 * time.Second}, nil
}

// method used only for unit test
func (s *WaitForShootClusterNewVersionStep) setInitialResourceVersionValue(initialResourceVersionValue string) {
	s.initialResourceVersion = initialResourceVersionValue
}
