package shootupgrade

import (
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type initialResourceVersions struct {
	versions map[string]string
}

func (irv *initialResourceVersions) add(operationID, resourceVersion string) {
	irv.versions[operationID] = resourceVersion
}

func (irv *initialResourceVersions) deleteFor(operationID string) {
	delete(irv.versions, operationID)
}

func (irv *initialResourceVersions) at(operationID string) string {
	v, _ := irv.versions[operationID]
	return v
}

func (irv *initialResourceVersions) find(operationID string) (string, bool) {
	v, ok := irv.versions[operationID]
	return v, ok
}

type WaitForShootClusterNewVersionStep struct {
	gardenerClient          GardenerClient
	nextStep                model.OperationStage
	timeLimit               time.Duration
	initialResourceVersions initialResourceVersions
}

func NewWaitForShootNewVersionStep(gardenerClient GardenerClient, nextStep model.OperationStage, timeLimit time.Duration) *WaitForShootClusterNewVersionStep {
	return &WaitForShootClusterNewVersionStep{
		gardenerClient:          gardenerClient,
		nextStep:                nextStep,
		timeLimit:               timeLimit,
		initialResourceVersions: initialResourceVersions{versions: make(map[string]string)},
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
		err := fmt.Errorf(fmt.Sprintf("Gardener Shoot cluster upgrade failed. Last Shoot state: %s, Shoot description: %s", shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description))
		return operations.StageResult{}, operations.NewNonRecoverableError(err)
	}

	v, ok := s.initialResourceVersions.find(operation.ID)
	if !ok {
		s.initialResourceVersions.add(operation.ID, shoot.ObjectMeta.ResourceVersion)
		return operations.StageResult{Stage: s.Name(), Delay: 20 * time.Second}, nil
	}

	logger.Info("Current resource version: ", shoot.ObjectMeta.ResourceVersion)

	if v != shoot.ObjectMeta.ResourceVersion {
		s.initialResourceVersions.deleteFor(operation.ID)
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	return operations.StageResult{Stage: s.Name(), Delay: 5 * time.Second}, nil
}
