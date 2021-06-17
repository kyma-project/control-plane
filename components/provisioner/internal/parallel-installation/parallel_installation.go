package parallel_installation

import (
	"sync"
	"time"

	"github.com/kyma-incubator/hydroform/install/installation"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/overrides"
	provisionerInstallation "github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

//go:generate mockery -name=PathFetcher
type PathFetcher interface {
	GetResourcePaths(string, []model.KymaComponentConfig) (string, string, error)
}

type (
	deployerCreator func(string, *config.Config, *overrides.Builder, func(string) func(deployment.ProcessUpdate)) KymaDeployer

	KymaDeployer interface {
		StartKymaDeployment() error
	}
)

type AsyncDeployment struct {
	deployer KymaDeployer
}

func (a AsyncDeployment) StartKymaDeployment(success func(), error func(error)) {
	go func() {
		err := a.deployer.StartKymaDeployment()
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
	log.Info("Installation complete")
}

type parallelInstallationService struct {
	createDeployer     deployerCreator
	pathFetcher        PathFetcher
	log                log.FieldLogger
	installationStatus map[string]*ComponentsStatus
	mux                *sync.Mutex
}

func NewParallelInstallationService(pathFetcher PathFetcher, creator deployerCreator, log log.FieldLogger) provisionerInstallation.Service {
	return &parallelInstallationService{
		log:                log,
		pathFetcher:        pathFetcher,
		createDeployer:     creator,
		mux:                &sync.Mutex{},
		installationStatus: make(map[string]*ComponentsStatus),
	}
}

func (p parallelInstallationService) getCallbackUpdate(runtimeID string) func(deployment.ProcessUpdate) {

	showCompStatus := func(comp components.KymaComponent) {
		if comp.Name != "" {
			p.log.Infof("Status of component '%s': %s", comp.Name, comp.Status)
		}
	}

	consumeEvent := func(event deployment.ProcessUpdate) {
		p.mux.Lock()
		p.installationStatus[runtimeID].ConsumeEvent(event)
		p.mux.Unlock()
	}

	return func(update deployment.ProcessUpdate) {
		switch update.Event {
		case deployment.ProcessStart:
			p.log.Infof("Starting installation phase '%s'", update.Phase)
			showCompStatus(update.Component)
		case deployment.ProcessRunning:
			showCompStatus(update.Component)
			consumeEvent(update)
		case deployment.ProcessFinished:
			p.log.Infof("Finished installation phase '%s' successfully", update.Phase)
		case deployment.ProcessExecutionFailure, deployment.ProcessForceQuitFailure, deployment.ProcessTimeoutFailure:
			p.log.Errorf("Installation failed on component %s, status: %s, error: %v", update.Component.Name, update.Event, update.Error)
			consumeEvent(update)
		default:
			//any other unknown case
			p.log.Infof("Unknown event: %s. The installation will continue", update.Event)
			showCompStatus(update.Component)
		}
	}
}

func (p parallelInstallationService) CheckInstallationState(runtimeID string, _ *rest.Config) (installation.InstallationState, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if v, found := p.installationStatus[runtimeID]; found {
		if v.IsFinished() {
			delete(p.installationStatus, runtimeID)
			p.log.Infof("installation for runtime %s successfully completed", runtimeID)
			return installation.InstallationState{
				State:       string(v1alpha1.StateInstalled),
				Description: v.StatusDescription(),
			}, nil
		}

		// installation process failed
		if err := v.InstallationError(); err != nil {
			delete(p.installationStatus, runtimeID)
			p.log.Infof("installation for runtime %s failed: %s", runtimeID, err)
			return installation.InstallationState{
					State: string(v1alpha1.StateError),
				}, installation.InstallationError{
					ShortMessage: "Installation failed",
				}
		}

		// installation component process failed, process will be repeated
		if err := v.ComponentError(); err != nil {
			p.log.Errorf("installation for runtime %s failed: %s", runtimeID, err)
			return installation.InstallationState{
					State: string(v1alpha1.StateError),
				}, installation.InstallationError{
					Recoverable:  true,
					ShortMessage: "Component installation failed",
				}
		}

		return installation.InstallationState{
			State:       string(v1alpha1.StateInProgress),
			Description: v.StatusDescription(),
		}, nil

	}

	return installation.InstallationState{State: installation.NoInstallationState}, nil
}

func (p parallelInstallationService) TriggerInstallation(runtimeID, kubeconfigRaw string, kymaProfile *model.KymaProfile, release model.Release, globalConfig model.Configuration, componentsConfig []model.KymaComponentConfig) error {
	p.log.Info("Installation triggered")

	// collect all necessary resources
	p.log.Info("Collect all require components")
	resourcePath, installationResourcePath, err := p.pathFetcher.GetResourcePaths(release.Version, componentsConfig)
	if err != nil {
		return errors.Wrap(err, "while collecting all components")
	}

	// prepare installation
	p.log.Info("Inject overrides")
	builder := &overrides.Builder{}
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
		Log:                           p.log.WithField("runtimeID", runtimeID),
		HelmMaxRevisionHistory:        10,
		Profile:                       string(*kymaProfile),
		ComponentList:                 ConvertToComponentList(componentsConfig),
		ResourcePath:                  resourcePath,
		InstallationResourcePath:      installationResourcePath,
		KubeconfigSource: config.KubeconfigSource{
			Content: kubeconfigRaw,
		},
		Version: release.Version,
	}

	// create Kyma deployer and start deployment
	p.log.Info("Start deployment process")
	deployer := p.createDeployer(runtimeID, installationCfg, builder, p.getCallbackUpdate)
	p.installationStatus[runtimeID] = NewComponentsStatus(componentsConfig)

	asyncDeployment := &AsyncDeployment{deployer}
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
