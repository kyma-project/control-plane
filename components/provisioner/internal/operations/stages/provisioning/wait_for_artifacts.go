package provisioning

import (
	"errors"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"

	"github.com/sirupsen/logrus"
)

//go:generate mockery -name=ResourceDownloader
type ResourceDownloader interface {
	Download(string, []model.KymaComponentConfig) error
}

type WaitForArtifactsStep struct {
	downloader ResourceDownloader
	nextStep   model.OperationStage
	timeLimit  time.Duration
}

func NewWaitForArtifactsStep(downloader ResourceDownloader, nextStep model.OperationStage, timeLimit time.Duration) *WaitForArtifactsStep {
	return &WaitForArtifactsStep{
		downloader: downloader,
		nextStep:   nextStep,
		timeLimit:  timeLimit,
	}
}

func (s *WaitForArtifactsStep) Name() model.OperationStage {
	return model.DownloadingArtifacts
}

func (s *WaitForArtifactsStep) TimeLimit() time.Duration {
	return s.timeLimit
}

func (s *WaitForArtifactsStep) Run(cluster model.Cluster, _ model.Operation, logger logrus.FieldLogger) (operations.StageResult, error) {
	components := cluster.KymaConfig.Components
	version := cluster.KymaConfig.Release.Version

	logger.Infof("Download all components for Kyma version: %s (%d components)", version, len(components))

	err := s.downloader.Download(version, components)
	if err != nil {
		logger.Errorf("failed to download components: %s", err)
		return operations.StageResult{}, errors.New("failed to download components")
	}

	return operations.StageResult{Stage: s.nextStep, Delay: 0}, nil
}
