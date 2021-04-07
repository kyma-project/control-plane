package installation

import (
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	"sync"
	"time"

	"github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/preinstaller"
	log "github.com/sirupsen/logrus"

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

type AsyncDeployment struct {
	*deployment.Deployment
}

func (a AsyncDeployment) StartKymaDeployment(success func (), error func(error) ) {
	go func(){
		err := a.Deployment.StartKymaDeployment()
		if err != nil {
			error(err)
		}
		success()
	}()
}

func callbackErrors(err error) {
	log.Error("Error during installation startup", err.Error())
}

func callbackSuccess() {
	log.Info("Installation started successfully")
}

type parallelInstallationService struct {
	downloader         ResourceDownloader
	log                logrus.FieldLogger
	installationStatus map[string]deployment.ProcessEvent
	mux                *sync.Mutex
}

func NewParallelInstallationService(downloader ResourceDownloader, log logrus.FieldLogger) Service {
	return &parallelInstallationService{
		log:                log,
		downloader:         downloader,
		mux:                &sync.Mutex{},
		installationStatus: make(map[string]deployment.ProcessEvent),
	}
}

func (p parallelInstallationService)getCallbackUpdate(runtimeID string) func (deployment.ProcessUpdate) {

	showCompStatus := func(comp components.KymaComponent) {
		if comp.Name != "" {
			log.Infof("Status of component '%s': %s", comp.Name, comp.Status)
		}
	}

	return func (update deployment.ProcessUpdate) {
		p.mux.Lock()
		p.installationStatus[runtimeID] = update.Event
		defer p.mux.Unlock()

		switch update.Event {
		case deployment.ProcessStart:
			log.Infof("Starting installation phase '%s'", update.Phase)
			p.installationStatus[runtimeID] = deployment.ProcessStart
		case deployment.ProcessRunning:
			p.installationStatus[runtimeID] = deployment.ProcessRunning
			showCompStatus(update.Component)
		case deployment.ProcessFinished:
			log.Infof("Finished installation phase '%s' successfully", update.Phase)
		default:
			//any failure case
			log.Infof("Process failed in phase '%s' with error state '%s':", update.Phase, update.Event)
			showCompStatus(update.Component)
		}
	}
}

func (p parallelInstallationService) CheckInstallationState(runtimeID string, _ *rest.Config) (installation.InstallationState, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if v, found := p.installationStatus[runtimeID]; found {
		if v == deployment.ProcessFinished {
			delete(p.installationStatus, runtimeID) // definetely should go to some cleanup function
			return installation.InstallationState{State: "Installed"}, nil
		}

		if v == deployment.ProcessExecutionFailure || v == deployment.ProcessForceQuitFailure || v == deployment.ProcessTimeoutFailure {
			return installation.InstallationState{State: "Error"}, installation.InstallationError{ShortMessage: "Error"}
		}
		return installation.InstallationState{State: "In progress"}, nil

	}
	return installation.InstallationState{State: installation.NoInstallationState}, nil
}

func (p parallelInstallationService) TriggerInstallation(runtimeID string, kubeconfigRaw *rest.Config, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error {
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
		return errors.Wrap(err, "while downloading components")
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

	p.log.Info("Start Kyma deployment")

	deployer, err := deployment.NewDeployment(installationCfg, builder, kubeClient, p.getCallbackUpdate(runtimeID))
	if err != nil {
		return errors.Wrap(err, "while creating deployer")
	}

	asyncDeployment := &AsyncDeployment{ deployer}
	asyncDeployment.StartKymaDeployment(callbackSuccess, callbackErrors)

	return nil
}

func (p parallelInstallationService) TriggerUpgrade(_ *rest.Config, _ *model.KymaProfile, _ model.Release, _ model.Configuration, _ []model.KymaComponentConfig) error {
	panic("TriggerUpgrade is not implemented")
}

func (p parallelInstallationService) TriggerUninstall(_ *rest.Config) error {
	panic("TriggerUninstall is not implemented")
}

func (p parallelInstallationService) PerformCleanup(_ *rest.Config) error {
	panic("PerformCleanup is not implemented")
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
