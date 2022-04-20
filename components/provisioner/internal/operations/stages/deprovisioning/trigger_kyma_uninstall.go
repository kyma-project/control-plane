package deprovisioning

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TriggerKymaUninstallStep struct {
	installationClient installation.Service
	gardenerClient     GardenerClient
	nextStep           model.OperationStage
	timeLimit          time.Duration
	delay              time.Duration
}

func NewTriggerKymaUninstallStep(gardenerClient GardenerClient, installationClient installation.Service, nextStep model.OperationStage, timeLimit time.Duration, delay time.Duration) *TriggerKymaUninstallStep {
	return &TriggerKymaUninstallStep{
		installationClient: installationClient,
		gardenerClient:     gardenerClient,
		nextStep:           nextStep,
		timeLimit:          timeLimit,
		delay:              delay,
	}
}

func (s *TriggerKymaUninstallStep) Name() model.OperationStage {
	return model.TriggerKymaUninstall
}

func (s *TriggerKymaUninstallStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *TriggerKymaUninstallStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

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

	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		err := fmt.Errorf("error: failed to create kubernetes config from raw: %s", err.Error())
		return operations.StageResult{}, operations.NewNonRecoverableError(util.K8SErrorToAppError(err))
	}

	err = s.installationClient.TriggerUninstall(k8sConfig)
	if err != nil {
		return operations.StageResult{}, apperrors.External(err.Error()).SetComponent(apperrors.ErrKymaInstaller).SetReason(apperrors.ErrTriggerKymaUninstall)
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}
