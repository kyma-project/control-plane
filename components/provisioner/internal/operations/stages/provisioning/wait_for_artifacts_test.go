package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWaitForArtifactsStep_Run(t *testing.T) {

	cluster := model.Cluster{
		KymaConfig: model.KymaConfig{
			Release: model.Release{
				Version: "1.2.3",
			},
			Components: componentsConfig(),
		},
	}
	nextStep := model.OperationStage("stageAfterDownload")

	t.Run("should return next step when finished", func(t *testing.T) {
		// given
		downloader := &mocks.ResourceDownloader{}
		downloader.On("Download", "1.2.3", componentsConfig()).Return(nil)

		stage := NewWaitForArtifactsStep(downloader, nextStep, time.Minute)

		// when
		result, err := stage.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.NoError(t, err)
		require.Equal(t, nextStep, result.Stage)
		require.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should return error when failed to download components", func(t *testing.T) {
		// given
		downloader := &mocks.ResourceDownloader{}
		downloader.On("Download", "1.2.3", componentsConfig()).Return(apperrors.Internal("error"))

		stage := NewWaitForArtifactsStep(downloader, nextStep, time.Minute)

		// when
		_, err := stage.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.Error(t, err)
	})
}

func componentsConfig() []model.KymaComponentConfig {
	return []model.KymaComponentConfig{
		{
			Component:     "core",
			Namespace:     "kyma-system",
			Configuration: model.Configuration{},
		},
		{
			Component:     "external",
			Namespace:     "kyma-system",
			SourceURL:     util.StringPtr("https://example.com"),
			Configuration: model.Configuration{},
		},
	}
}
