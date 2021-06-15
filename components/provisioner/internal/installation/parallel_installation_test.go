package installation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/installation/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	installationConfig "github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/overrides"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	runtimeIDOne        = "8f831ae1-92ac-4a51-b772-f638cd7c66f4"
	runtimeIDTwo        = "cb6bdc16-d0db-40c3-97e3-66d0650c09f2"
	resourcePath        = "/path/to/components"
	installResourcePath = "path/to/externals"
)

func TestParallelInstallationService_TriggerInstallation(t *testing.T) {
	t.Run("successful installation", func(t *testing.T) {
		// given
		kymaRelease := model.Release{Version: "1.20.0"}
		profile := model.KymaProfile("test")

		downloader := &mocks.ResourceDownloader{}
		downloader.On("Download", kymaRelease.Version, componentsConfig("test#1")).Return(resourcePath, installResourcePath, nil)
		downloader.On("Download", kymaRelease.Version, componentsConfig("test#2")).Return(resourcePath, installResourcePath, nil)

		createDeployer := func(rID string, cfg *installationConfig.Config, ob *overrides.Builder, callback func(string) func(deployment.ProcessUpdate)) KymaDeployer {
			return &fakeDeployer{ID: rID, cfg: cfg, callback: callback}
		}

		svc := NewParallelInstallationService(downloader, createDeployer, logrus.New())

		// when
		for rID, namespace := range map[string]string{runtimeIDOne: "test#1", runtimeIDTwo: "test#2"} {
			err := svc.TriggerInstallation(rID, "", &profile, kymaRelease, globalConfig(), componentsConfig(namespace))
			require.NoError(t, err)
		}

		// then
		for _, runtimeID := range []string{runtimeIDOne, runtimeIDTwo} {
			err := wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (done bool, err error) {
				state, err := svc.CheckInstallationState(runtimeID, nil)
				require.NoError(t, err)
				return state.State == string(v1alpha1.StateInstalled), nil
			})
			require.NoError(t, err)
		}
	})

	t.Run("installation failed", func(t *testing.T) {
		// given
		kymaRelease := model.Release{Version: "1.20.0"}
		profile := model.KymaProfile("test")

		downloader := &mocks.ResourceDownloader{}
		downloader.On("Download", kymaRelease.Version, componentsConfig("test")).Return(resourcePath, installResourcePath, nil)

		createDeployer := func(rID string, cfg *installationConfig.Config, ob *overrides.Builder, callback func(string) func(deployment.ProcessUpdate)) KymaDeployer {
			return &fakeDeployer{ID: rID, cfg: cfg, callback: callback, shouldFail: true}
		}

		svc := NewParallelInstallationService(downloader, createDeployer, logrus.New())

		// when
		err := svc.TriggerInstallation(runtimeIDOne, "", &profile, kymaRelease, globalConfig(), componentsConfig("test"))
		require.NoError(t, err)

		//err = wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		err = wait.PollImmediate(100*time.Millisecond, 1*time.Second, func() (done bool, err error) {
			state, err := svc.CheckInstallationState(runtimeIDOne, nil)
			if err != nil {
				installErr := installationSDK.InstallationError{}
				require.True(t, errors.As(err, &installErr))
				return state.State == string(v1alpha1.StateError) && !installErr.Recoverable, nil
			}

			// fake installation in progress
			return false, nil
		})
		require.NoError(t, err)
	})
}

func componentsConfig(namespace string) []model.KymaComponentConfig {
	return []model.KymaComponentConfig{
		{
			Component:     "cluster-essentials",
			Namespace:     namespace,
			Configuration: model.Configuration{},
		},
		{
			Component:     "core",
			Namespace:     namespace,
			Configuration: model.Configuration{},
		},
		{
			Component:     "rafter",
			Namespace:     namespace,
			Configuration: model.Configuration{ConfigEntries: make([]model.ConfigEntry, 0, 0)},
		},
		{
			Component:     "external",
			Namespace:     namespace,
			SourceURL:     util.StringPtr("https://example.com"),
			Configuration: model.Configuration{},
		},
	}
}

func globalConfig() model.Configuration {
	return model.Configuration{
		ConflictStrategy: gqlschema.ConflictStrategyReplace.String(),
		ConfigEntries: []model.ConfigEntry{
			model.NewConfigEntry("global.config.key", "globalValue", false),
			model.NewConfigEntry("global.config.key2", "globalValue2", false),
			model.NewConfigEntry("global.secret.key", "globalSecretValue", true),
		}}
}

type fakeDeployer struct {
	ID         string
	cfg        *installationConfig.Config
	callback   func(string) func(deployment.ProcessUpdate)
	shouldFail bool
}

func (fd *fakeDeployer) StartKymaDeployment() error {
	if fd.cfg.ResourcePath != resourcePath || fd.cfg.InstallationResourcePath != installResourcePath {
		return fmt.Errorf("path to resources is incorrect")
	}

	installationEvent := deployment.ProcessRunning
	componentEvent := components.StatusInstalled
	if fd.shouldFail {
		installationEvent = deployment.ProcessExecutionFailure
		componentEvent = components.StatusError
	}

	for _, cmp := range fd.cfg.ComponentList.Components {
		f := fd.callback(fd.ID)
		f(deployment.ProcessUpdate{
			Event: installationEvent,
			Phase: deployment.InstallComponents,
			Component: components.KymaComponent{
				Name:      cmp.Name,
				Namespace: cmp.Namespace,
				Status:    componentEvent,
			},
		})
	}

	if fd.shouldFail {
		return fmt.Errorf("deployment failed")
	}
	return nil
}
