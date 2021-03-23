package installation

import (
	"fmt"
	"sync"
	"time"

	"github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/preinstaller"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/avast/retry-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ResourceDownloader interface {
	Download(string, []model.KymaComponentConfig) (string, string, error)
}

type parallelInstallationService struct {
	downloader ResourceDownloader
	log        logrus.FieldLogger
}

func NewParallelInstallationService(downloader ResourceDownloader, log logrus.FieldLogger) Service {
	return &parallelInstallationService{
		log:        log,
		downloader: downloader,
	}
}

func (p parallelInstallationService) CheckInstallationState(kubeconfig *rest.Config) (installation.InstallationState, error) {
	return installation.InstallationState{State: "Installed"}, nil
}

func (p parallelInstallationService) TriggerInstallation(kubeconfigRaw *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error {
	p.log.Info("Installation triggered")

	kubeClient, err := kubernetes.NewForConfig(kubeconfigRaw)
	if err != nil {
		return errors.Wrap(err, "while creating kubernetes client")
	}

	dynamicClient, err := dynamic.NewForConfig(kubeconfigRaw)
	if err != nil {
		return errors.Wrap(err, "while creating dynamic client")
	}

	// download all resources
	p.log.Info("Collect all require components")
	resourcePath, installationResourcePath, err := p.downloader.Download(release.Version, componentsConfig)
	if err != nil {
		return errors.Wrap(err, "while collecting components for Kyma")
	}

	// prepare installation
	p.log.Info("Inject overrides")
	builder := &deployment.OverridesBuilder{}
	err = SetOverrides(builder, componentsConfig, globalConfig)
	if err != nil {
		return errors.Wrap(err, "while set overrides to the OverridesBuilder")
	}

	installationCfg := &config.Config{
		WorkersCount:                  4,
		CancelTimeout:                 20 * time.Minute,
		QuitTimeout:                   25 * time.Minute,
		HelmTimeoutSeconds:            60 * 8,
		BackoffInitialIntervalSeconds: 3,
		BackoffMaxElapsedTimeSeconds:  60 * 5,
		Log:                           logrus.New(),
		HelmMaxRevisionHistory:        10,
		Profile:                       string(*kymaProfile),
		ComponentList:                 ConvertToComponentList(componentsConfig),
		ResourcePath:                  resourcePath,
		InstallationResourcePath:      installationResourcePath,
		Version:                       release.Version,
	}

	commonRetryOpts := []retry.Option{
		retry.Delay(time.Duration(installationCfg.BackoffInitialIntervalSeconds) * time.Second),
		retry.Attempts(uint(installationCfg.BackoffMaxElapsedTimeSeconds / installationCfg.BackoffInitialIntervalSeconds)),
		retry.DelayType(retry.FixedDelay),
	}

	preInstallerCfg := preinstaller.Config{
		InstallationResourcePath: installationCfg.InstallationResourcePath,
		Log:                      installationCfg.Log,
	}

	resourceParser := &preinstaller.GenericResourceParser{}
	resourceManager := preinstaller.NewDefaultResourceManager(dynamicClient, preInstallerCfg.Log, commonRetryOpts)
	resourceApplier := preinstaller.NewGenericResourceApplier(installationCfg.Log, resourceManager)
	preInstaller := preinstaller.NewPreInstaller(resourceApplier, resourceParser, preInstallerCfg, dynamicClient, commonRetryOpts)

	// Install CRDs and create namespace
	p.log.Info("Install CRDs")
	result, err := preInstaller.InstallCRDs()
	if err != nil || len(result.NotInstalled) > 0 {
		return errors.Wrap(err, "while installing CRDs")
	}

	p.log.Info("Create installation namespace")
	result, err = preInstaller.CreateNamespaces()
	if err != nil || len(result.NotInstalled) > 0 {
		return errors.Wrap(err, "while creating namespace")
	}

	// Install Kyma
	p.log.Info("Start Kyma deployment")
	progressCh := make(chan deployment.ProcessUpdate)
	deployer, err := deployment.NewDeployment(installationCfg, builder, kubeClient, progressCh)
	if err != nil {
		return errors.Wrap(err, "while creating deployer")
	}

	go func() {
		for update := range progressCh {
			switch update.Event {
			case deployment.ProcessStart:
				p.log.Info("installation process started")
			case deployment.ProcessRunning:
				p.log.Info("installation process in progress...")
			case deployment.ProcessFinished:
				p.log.Info("installation process succeeded")
			default:
				//any failure case
				p.log.Errorf("process failed in phase '%s' with error state '%s'", update.Phase, update.Event)
			}
		}
	}()

	// Start Kyma deployment
	var processError error
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err = deployer.StartKymaDeployment()
		defer wg.Done()
		if err != nil {
			processError = fmt.Errorf("starting Kyma deployment failed: %s", err)
		}
	}()
	wg.Wait()

	return processError
}

func (p parallelInstallationService) TriggerUpgrade(_ *rest.Config, _ *model.KymaProfile, _ model.Release, _ model.Configuration, _ []model.KymaComponentConfig) error {
	panic("TriggerUpgrade is not implemented")
}

func (p parallelInstallationService) TriggerUninstall(_ *rest.Config) error {
	panic("TriggerUninstall is not implemented")
}

func (p parallelInstallationService) PerformCleanup(_ *rest.Config) error {
	panic("PerformCleanup is not implemented ")
}

func ConvertToComponentList(components []model.KymaComponentConfig) *config.ComponentList {
	var list config.ComponentList

	for _, component := range components {
		if component.Prerequisite {
			list.Prerequisites = append(list.Prerequisites, config.ComponentDefinition{
				Name:      string(component.Component),
				Namespace: component.Namespace,
			})
			continue
		}
		list.Components = append(list.Components, config.ComponentDefinition{
			Name:      string(component.Component),
			Namespace: component.Namespace,
		})
	}

	return &list
}
