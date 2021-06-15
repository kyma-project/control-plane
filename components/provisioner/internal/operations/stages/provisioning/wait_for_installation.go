package provisioning

import (
	"errors"
	"fmt"
	"time"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/sirupsen/logrus"
)

type WaitForInstallationStep struct {
	installationClients map[model.KymaInstaller]installation.Service
	nextStep            model.OperationStage
	timeLimit           time.Duration
	dbSession           dbsession.WriteSession
}

func NewWaitForInstallationStep(
	installationClients map[model.KymaInstaller]installation.Service,
	nextStep model.OperationStage,
	timeLimit time.Duration,
	dbSession dbsession.WriteSession,
) *WaitForInstallationStep {
	return &WaitForInstallationStep{
		installationClients: installationClients,
		nextStep:            nextStep,
		timeLimit:           timeLimit,
		dbSession:           dbSession,
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
		return operations.StageResult{}, fmt.Errorf("error: failed to create kubernetes config from raw: %s", err.Error())
	}

	client, ok := s.installationClients[cluster.KymaConfig.Installer]
	if !ok {
		return operations.StageResult{}, fmt.Errorf("error: installation client for installation %s does not exist", cluster.KymaConfig.Installer)
	}

	installationState, err := client.CheckInstallationState(cluster.ID, k8sConfig)
	if err != nil {
		installErr := installationSDK.InstallationError{}
		if errors.As(err, &installErr) {
			message := fmt.Sprintf("Installation error occurred: %s", installErr.Error())
			logger.Warn(message)
			s.saveInstallationState(message, logger, operation)
			if installErr.Recoverable {
				return operations.StageResult{Stage: s.Name(), Delay: 30 * time.Second}, nil
			}

			return operations.StageResult{}, operations.NewNonRecoverableError(err)
		}

		return operations.StageResult{}, fmt.Errorf("error: failed to check installation state: %s", err.Error())
	}

	if installationState.State == string(v1alpha1.StateInstalled) {
		message := fmt.Sprintf("Installation completed: %s", installationState.Description)
		logger.Info(message)
		s.saveInstallationState(message, logger, operation)
		return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
	}

	if installationState.State == installationSDK.NoInstallationState {
		return operations.StageResult{}, fmt.Errorf("installation not yet started")
	}

	message := fmt.Sprintf("Installation in progress: %s", installationState.Description)
	logger.Info(message)
	s.saveInstallationState(message, logger, operation)
	return operations.StageResult{Stage: s.Name(), Delay: 30 * time.Second}, nil
}

func (s *WaitForInstallationStep) saveInstallationState(message string, logger logrus.FieldLogger, operation model.Operation) {
	dberr := s.dbSession.UpdateOperationState(operation.ID, message, operation.State, time.Now())
	if dberr != nil {
		logger.Errorf("error updating installation state: %s", dberr.Error())
	}
}
