package installation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/util"

	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	pkgErrors "github.com/pkg/errors"

	"github.com/kyma-incubator/hydroform/install/installation"
	"k8s.io/client-go/rest"
)

const (
	tillerWaitTime = 10 * time.Minute

	installAction   = "installation"
	upgradeAction   = "upgrade"
	replaceStrategy = "Replace"
)

type InstallationHandler func(*rest.Config, ...installation.InstallationOption) (installation.Installer, error)

//go:generate mockery -name=Service
type Service interface {
	CheckInstallationState(kubeconfig *rest.Config) (installation.InstallationState, error)
	TriggerInstallation(kubeconfigRaw *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error
	TriggerUpgrade(kubeconfigRaw *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error
	TriggerUninstall(kubeconfig *rest.Config) error
	PerformCleanup(kubeconfig *rest.Config) error
}

func NewInstallationService(installationTimeout time.Duration, installationHandler InstallationHandler, clusterCleanupResourceSelector string) Service {
	return &installationService{
		kymaInstallationTimeout:        installationTimeout,
		installationHandler:            installationHandler,
		clusterCleanupResourceSelector: clusterCleanupResourceSelector,
	}
}

type installationService struct {
	kymaInstallationTimeout        time.Duration
	installationHandler            InstallationHandler
	clusterCleanupResourceSelector string
}

func (s *installationService) PerformCleanup(kubeconfig *rest.Config) error {
	cli, err := NewServiceCatalogCleanupClient(kubeconfig)
	if err != nil {
		return err
	}
	return cli.PerformCleanup(s.clusterCleanupResourceSelector)
}

func (s *installationService) TriggerInstallation(kubeconfig *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error {
	kymaInstaller, err := s.createKymaInstaller(kubeconfig, kymaProfile, componentsConfig)
	if err != nil {
		return fmt.Errorf("failed to trigger installation: %s", err.Error())
	}
	return s.triggerAction(release, globalConfig, componentsConfig, kymaInstaller, kymaInstaller.PrepareInstallation, installAction)
}

func (s *installationService) TriggerUpgrade(kubeconfig *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error {
	kymaInstaller, err := s.createKymaInstaller(kubeconfig, kymaProfile, componentsConfig)
	if err != nil {
		return fmt.Errorf("failed to trigger upgrade: %s", err.Error())
	}
	return s.triggerAction(release, globalConfig, componentsConfig, kymaInstaller, kymaInstaller.PrepareUpgrade, upgradeAction)
}

func (s *installationService) triggerAction(
	release model.Release,
	globalConfig model.Configuration,
	componentsConfig []model.KymaComponentConfig,
	installer installation.Installer,
	prepareFunction func(installation.Installation) error,
	actionName string) error {

	installationConfig := installation.Installation{
		TillerYaml:    release.TillerYAML,
		InstallerYaml: release.InstallerYAML,
		Configuration: NewInstallationConfiguration(globalConfig, componentsConfig),
	}

	if err := prepareFunction(installationConfig); err != nil {
		return pkgErrors.Wrap(err, fmt.Sprintf("Failed to prepare %s", actionName))
	}

	installationCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// We are not waiting for events, just triggering installation
	if _, _, err := installer.StartInstallation(installationCtx); err != nil {
		return pkgErrors.Wrap(err, fmt.Sprintf("Failed to start Kyma %s", actionName))
	}

	return nil
}

func (s *installationService) createKymaInstaller(kubeconfig *rest.Config, kymaProfile *model.KymaProfile, componentsConfig []model.KymaComponentConfig) (installation.Installer, error) {
	kymaInstaller, err := s.installationHandler(
		kubeconfig,
		installation.WithTillerWaitTime(tillerWaitTime),
		installation.WithInstallationCRModification(GetInstallationCRModificationFunc(kymaProfile, componentsConfig)),
	)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "Failed to create Kyma installer")
	}

	return kymaInstaller, nil
}

func (s *installationService) CheckInstallationState(kubeconfig *rest.Config) (installation.InstallationState, error) {
	return installation.CheckInstallationState(kubeconfig)
}

func (s *installationService) TriggerUninstall(kubeconfig *rest.Config) error {
	return installation.TriggerUninstall(kubeconfig)
}

func (s *installationService) waitForInstallation(runtimeId string, stateChannel <-chan installation.InstallationState, errorChannel <-chan error) error {
	for {
		select {
		case state, ok := <-stateChannel:
			if !ok {
				return nil
			}
			logrus.Infof("Installing Kyma on Runtime %s. Description: %s, State: %s", runtimeId, state.Description, state.State)
		case err, ok := <-errorChannel:
			if !ok {
				continue
			}

			installationError := installation.InstallationError{}
			if ok := errors.As(err, &installationError); ok {
				logrus.Warnf("Warning: installation error occurred while installing Kyma for %s Runtime: %s. Details: %s", runtimeId, installationError.Error(), installationError.Details())
				continue
			}

			return fmt.Errorf("an error occurred while installing Kyma for %s Runtime: %s.", runtimeId, err.Error())
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func GetInstallationCRModificationFunc(kymaProfile *model.KymaProfile, componentsConfig []model.KymaComponentConfig) func(*v1alpha1.Installation) {
	return func(installation *v1alpha1.Installation) {
		if kymaProfile != nil {
			installation.Spec.Profile = toKymaProfile(*kymaProfile)
		}

		components := make([]v1alpha1.KymaComponent, 0, len(componentsConfig))

		for _, cc := range componentsConfig {
			components = append(components, v1alpha1.KymaComponent{
				Name:      string(cc.Component),
				Namespace: cc.Namespace,
				Source:    toKymaComponentSource(util.UnwrapStr(cc.SourceURL)),
			})
		}

		installation.Spec.Components = components
	}
}

func toKymaProfile(profile model.KymaProfile) v1alpha1.KymaProfile {
	switch profile {
	case model.ProductionProfile:
		return v1alpha1.ProductionProfile
	case model.EvaluationProfile:
		return v1alpha1.EvaluationProfile
	default:
		return v1alpha1.KymaProfile("")
	}
}

func toKymaComponentSource(sourceURL string) *v1alpha1.ComponentSource {
	if sourceURL == "" {
		return nil
	}
	return &v1alpha1.ComponentSource{URL: sourceURL}
}

func NewInstallationConfiguration(globalConfg model.Configuration, componentsConfig []model.KymaComponentConfig) installation.Configuration {
	installationConfig := installation.Configuration{
		Configuration:          make([]installation.ConfigEntry, 0, len(globalConfg.ConfigEntries)),
		ComponentConfiguration: make([]installation.ComponentConfiguration, 0, len(componentsConfig)),
	}

	installationConfig.Configuration = toInstallationConfigEntries(globalConfg.ConfigEntries)
	installationConfig.ConflictStrategy = replaceStrategy

	for _, componentCfg := range componentsConfig {
		installationComponentConfig := installation.ComponentConfiguration{
			Component:     string(componentCfg.Component),
			Configuration: toInstallationConfigEntries(componentCfg.Configuration.ConfigEntries),
		}

		installationComponentConfig.ConflictStrategy = replaceStrategy

		installationConfig.ComponentConfiguration = append(installationConfig.ComponentConfiguration, installationComponentConfig)
	}

	return installationConfig
}

func toInstallationConfigEntries(entries []model.ConfigEntry) []installation.ConfigEntry {
	installationCfgEntries := make([]installation.ConfigEntry, 0, len(entries))

	for _, e := range entries {
		installationCfgEntries = append(installationCfgEntries, toInstallationConfigEntry(e))
	}

	return installationCfgEntries
}

func toInstallationConfigEntry(entry model.ConfigEntry) installation.ConfigEntry {
	return installation.ConfigEntry{
		Key:    entry.Key,
		Value:  entry.Value,
		Secret: entry.Secret,
	}
}
