package provisioning

import (
	"errors"
	"fmt"
	"time"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	pkgErrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type InstallKymaStep struct {
	installationClient installation.Service
	nextStep           model.OperationStage
	timeLimit          time.Duration
}

func NewInstallKymaStep(installationClient installation.Service, nextStep model.OperationStage, timeLimit time.Duration) *InstallKymaStep {
	return &InstallKymaStep{
		installationClient: installationClient,
		nextStep:           nextStep,
		timeLimit:          timeLimit,
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

	k8sConfig, err := k8s.ParseToK8sConfig([]byte(*cluster.Kubeconfig))
	if err != nil {
		return operations.StageResult{}, util.K8SErrorToAppError(pkgErrors.Wrap(err, "error: failed to create kubernetes config from raw")).SetComponent(apperrors.ErrClusterK8SClient)
	}

	installationState, err := s.installationClient.CheckInstallationState(k8sConfig)
	if err != nil {
		installErr := installationSDK.InstallationError{}
		if errors.As(err, &installErr) {
			logger.Warnf("Installation already in progress, proceeding to next step...")
			return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
		}

		return operations.StageResult{}, apperrors.External(fmt.Sprintf("error: failed to check installation state: %s", err.Error())).SetComponent(apperrors.ErrKymaInstaller).SetReason(apperrors.ErrCheckKymaInstallationState)
	}

	if installationState.State != installationSDK.NoInstallationState && installationState.State != string(v1alpha1.StateEmpty) {
		logger.Warnf("Installation already in progress, proceeding to next step...")
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	err = s.installationClient.TriggerInstallation(
		k8sConfig,
		cluster.KymaConfig.Profile,
		cluster.KymaConfig.Release,
		cluster.KymaConfig.GlobalConfiguration,
		cluster.KymaConfig.Components)
	if err != nil {
		return operations.StageResult{}, apperrors.External(fmt.Sprintf("error: failed to start installation: %s", err.Error())).SetComponent(apperrors.ErrKymaInstaller).SetReason(apperrors.ErrTriggerKymaInstallation)
	}

	logger.Warnf("Installation started, proceeding to next step...")
	return operations.StageResult{Stage: s.nextStep, Delay: 30 * time.Second}, nil
}
