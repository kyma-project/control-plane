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
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	pkgErrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type WaitForInstallationStep struct {
	installationClient installation.Service
	nextStep           model.OperationStage
	timeLimit          time.Duration
	dbSession          dbsession.WriteSession
}

func NewWaitForInstallationStep(installationClient installation.Service, nextStep model.OperationStage, timeLimit time.Duration, dbSession dbsession.WriteSession) *WaitForInstallationStep {
	return &WaitForInstallationStep{
		installationClient: installationClient,
		nextStep:           nextStep,
		timeLimit:          timeLimit,
		dbSession:          dbSession,
	}
}

func (s *WaitForInstallationStep) Name() model.OperationStage {
	return model.WaitingForInstallation
}

func (s *WaitForInstallationStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForInstallationStep) Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {

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
			message := fmt.Sprintf("Installation error occurred: %s", installErr.Error())
			logger.Warn(message)
			s.saveInstallationState(message, logger, operation)
			if installErr.Recoverable {
				return operations.StageResult{Stage: s.Name(), Delay: 30 * time.Second}, nil
			}

			reason := util.KymaInstallationErrorToErrReason(installErr.ErrorEntries...)

			return operations.StageResult{}, operations.NewNonRecoverableError(apperrors.External(installErr.Error()).SetComponent(apperrors.ErrKymaInstaller).SetReason(reason))
		}

		return operations.StageResult{}, apperrors.External(fmt.Sprintf("error: failed to check installation state: %s", err.Error())).SetComponent(apperrors.ErrKymaInstaller).SetReason(apperrors.ErrCheckKymaInstallationState)
	}

	if installationState.State == string(v1alpha1.StateInstalled) {
		message := fmt.Sprintf("Installation completed: %s", installationState.Description)
		logger.Info(message)
		s.saveInstallationState(message, logger, operation)
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	if installationState.State == installationSDK.NoInstallationState {
		return operations.StageResult{}, apperrors.External("installation not yet started").SetComponent(apperrors.ErrKymaInstaller).SetReason(apperrors.ErrReason(installationSDK.NoInstallationState))
	}

	message := fmt.Sprintf("Installation in progress: %s", installationState.Description)
	logger.Info(message)
	s.saveInstallationState(message, logger, operation)
	return operations.StageResult{Stage: s.Name(), Delay: 30 * time.Second}, nil
}

func (s *WaitForInstallationStep) saveInstallationState(message string, logger logrus.FieldLogger, operation model.Operation) {
	dberr := s.dbSession.UpdateOperationState(operation.ID, message, operation.State, time.Now())
	if dberr != nil {
		logger.Warnf("error updating installation state: %s", dberr.Error())
	}
}
