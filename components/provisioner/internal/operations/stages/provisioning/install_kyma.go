package provisioning

import (
	"errors"
	"fmt"
	"time"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/sirupsen/logrus"
)

type InstallKymaStep struct {
	installationClients map[model.KymaInstaller]installation.Service
	nextStep            model.OperationStage
	timeLimit           time.Duration
}

func NewInstallKymaStep(installationClients map[model.KymaInstaller]installation.Service, nextStep model.OperationStage, timeLimit time.Duration) *InstallKymaStep {
	return &InstallKymaStep{
		installationClients: installationClients,
		nextStep:            nextStep,
		timeLimit:           timeLimit,
	}
}

func (s *InstallKymaStep) Name() model.OperationStage {
	return model.StartingInstallation
}

func (s *InstallKymaStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *InstallKymaStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {
	if cluster.Kubeconfig == nil {
		return operations.StageResult{}, fmt.Errorf("error: kubeconfig is nil")
	}

	client, ok := s.installationClients[cluster.KymaConfig.Installer]
	if !ok {
		return operations.StageResult{}, fmt.Errorf("error: installation client for installation %s does not exist", cluster.KymaConfig.Installer)
	}

	if cluster.KymaConfig.Installer == model.ParallelInstaller {
		return s.runParallelInstall(client, cluster, logger)
	}
	return s.runKymaOperatorInstall(client, cluster, logger)
}

func (s *InstallKymaStep) runKymaOperatorInstall(client installation.Service, cluster model.Cluster, logger logrus.FieldLogger) (operations.StageResult, error) {
	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("error: failed to create kubernetes config from raw: %s", err.Error())
	}

	installationState, err := client.CheckInstallationState(cluster.ID, k8sConfig)
	if err != nil {
		installErr := installationSDK.InstallationError{}
		if errors.As(err, &installErr) {
			logger.Warnf("Installation already in progress, proceeding to next step...")
			return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
		}

		return operations.StageResult{}, fmt.Errorf("error: failed to check installation state: %s", err.Error())
	}

	if installationState.State != installationSDK.NoInstallationState && installationState.State != string(v1alpha1.StateEmpty) {
		logger.Warnf("Installation already in progress, proceeding to next step...")
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	err = client.TriggerInstallation(
		cluster.ID,
		*cluster.Kubeconfig,
		cluster.KymaConfig.Profile,
		cluster.KymaConfig.Release,
		cluster.KymaConfig.GlobalConfiguration,
		cluster.KymaConfig.Components)
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("error: failed to start installation: %s", err.Error())
	}

	logger.Warnf("Installation started, proceeding to next step...")
	return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
}

func (s *InstallKymaStep) runParallelInstall(client installation.Service, cluster model.Cluster, logger logrus.FieldLogger) (operations.StageResult, error) {
	err := client.TriggerInstallation(
		cluster.ID,
		*cluster.Kubeconfig,
		cluster.KymaConfig.Profile,
		cluster.KymaConfig.Release,
		cluster.KymaConfig.GlobalConfiguration,
		cluster.KymaConfig.Components)
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("error: failed to start installation: %s", err.Error())
	}

	logger.Warnf("Installation started, proceeding to next step...")
	return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
}
