package upgrade

import (
	"errors"
	"fmt"
	"time"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/sirupsen/logrus"
)

type UpgradeKymaStep struct {
	installationClient installation.Service
	nextStep           model.OperationStage
	timeLimit          time.Duration
}

type InstallComponentStep struct {
	installationClient installation.Service
	timeLimit          time.Duration
}

var _ operations.Step = &InstallComponentStep{}

func NewInstallComponentKymaStep(installationClient installation.Service, timeLimit time.Duration) *InstallComponentStep {
	return &InstallComponentStep{
		installationClient: installationClient,
		timeLimit:          timeLimit,
	}
}

func NewUpgradeKymaStep(installationClient installation.Service, nextStep model.OperationStage, timeLimit time.Duration) *UpgradeKymaStep {
	return &UpgradeKymaStep{
		installationClient: installationClient,
		nextStep:           nextStep,
		timeLimit:          timeLimit,
	}
}

func (s *UpgradeKymaStep) Name() model.OperationStage {
	return model.StartingUpgrade
}

func (s *UpgradeKymaStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *UpgradeKymaStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

	if cluster.Kubeconfig == nil {
		return operations.StageResult{}, fmt.Errorf("error: kubeconfig is nil")
	}

	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("error: failed to create kubernetes config from raw: %s", err.Error())
	}

	installationState, err := s.installationClient.CheckInstallationState(k8sConfig)
	if err != nil {
		installErr := installationSDK.InstallationError{}
		if errors.As(err, &installErr) {
			if installErr.Recoverable {
				logger.Warnf("Upgrade already in progress, proceeding to next step...")
				return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
			}

			logger.Warnf("Installation is in unrecoverable error state, triggering the upgrade ...")
			installationState.State = "Error"
		} else {
			return operations.StageResult{}, fmt.Errorf("error: failed to check installation CR state: %s", err.Error())
		}
	}

	if installationState.State == installationSDK.NoInstallationState {
		return operations.StageResult{}, operations.NewNonRecoverableError(fmt.Errorf("error: Installation CR not found in the cluster, cannot trigger upgrade"))
	}

	if installationState.State == "Installed" || installationState.State == "Error" {
		err = s.installationClient.TriggerUpgrade(
			k8sConfig,
			cluster.KymaConfig.Profile,
			cluster.KymaConfig.Release,
			cluster.KymaConfig.GlobalConfiguration,
			cluster.KymaConfig.Components)
		if err != nil {
			return operations.StageResult{}, fmt.Errorf("error: failed to trigger upgrade: %s", err.Error())
		}
	}

	if installationState.State == "InProgress" {
		logger.Warnf("Upgrade already in progress, proceeding to next step...")
		return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
}

func (s *InstallComponentStep) Name() model.OperationStage {
	return model.InstallingComponent
}

func (s *InstallComponentStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *InstallComponentStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {
	/*if cluster.Kubeconfig == nil {
		return operations.StageResult{}, fmt.Errorf("error: kubeconfig is nil")
	}

	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		return operations.StageResult{}, fmt.Errorf("error: failed to create kubernetes config from raw: %s", err.Error())
	}

	installationState, err := s.installationClient.CheckInstallationState(k8sConfig)
	if err != nil {
		installErr := installationSDK.InstallationError{}
		if errors.As(err, &installErr) {
			if installErr.Recoverable {
				logger.Warnf("Upgrade already in progress, proceeding to next step...")
				return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
			}

			logger.Warnf("Installation is in unrecoverable error state, triggering the upgrade ...")
			installationState.State = "Error"
		} else {
			return operations.StageResult{}, fmt.Errorf("error: failed to check installation CR state: %s", err.Error())
		}
	}

	if installationState.State == installationSDK.NoInstallationState {
		return operations.StageResult{}, operations.NewNonRecoverableError(fmt.Errorf("error: Installation CR not found in the cluster, cannot trigger upgrade"))
	}

	if installationState.State == "Installed" {
		err = s.installationClient.TriggerUpgrade(
			k8sConfig,
			cluster.KymaConfig.Profile,
			cluster.KymaConfig.Release,
			cluster.KymaConfig.GlobalConfiguration,
			cluster.KymaConfig.Components)
		if err != nil {
			return operations.StageResult{}, fmt.Errorf("error: failed to trigger upgrade: %s", err.Error())
		}
	}

	if installationState.State == "InProgress" {
		logger.Warnf("Upgrade already in progress, proceeding to next step...")
		return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
	}
	*/
	return operations.StageResult{}, nil
}
