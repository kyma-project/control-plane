package shootupgrade

import (
	"time"

	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GardenerClient interface {
	Get(name string, options v1.GetOptions) (*gardener_types.Shoot, error)
}

type WaitForShootClusterNewVersion struct {
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

func (s WaitForShootClusterNewVersion) Name() model.OperationStage {
	return model.WaitingForShootNewVersion
}

func (s *WaitForShootClusterNewVersion) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForShootClusterNewVersion) Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

	gardenerConfig := cluster.ClusterConfig

	shoot, err := s.gardenerClient.Get(gardenerConfig.Name, v1.GetOptions{})
	if err != nil {
		return operations.StageResult{}, err
	}

	if s.initialResourceVersion == "" {
		s.initialResourceVersion = shoot.ObjectMeta.ResourceVersion
		return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
	}

	if s.initialResourceVersion != shoot.ObjectMeta.ResourceVersion {
		logger.Info("Shoot upgrade operation has generated new resource version")
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}


	/*if lastOperation != nil {

		if lastOperation.State == gardencorev1beta1.LastOperationStateSucceeded {
			logger.Info("Shoot upgrade operation has completed successfully")
			return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
		}

		if lastOperation.State == gardencorev1beta1.LastOperationStateFailed {
			logger.Warningf("Gardener Shoot cluster upgrade operation failed! Last state: %s, Description: %s", lastOperation.State, lastOperation.Description)

			err := errors.New(fmt.Sprintf("Gardener Shoot cluster upgrade failed. Last Shoot state: %s, Shoot description: %s", lastOperation.State, lastOperation.Description))

			return operations.StageResult{}, operations.NewNonRecoverableError(err)
		}
	}*/

	return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
}
