package parallel_installation

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
)

type ComponentsStatus struct {
	components         map[string]bool
	componentFailed    error
	installationFailed error
}

func NewComponentsStatus(components []model.KymaComponentConfig) *ComponentsStatus {
	cmpList := make(map[string]bool)
	for _, component := range components {
		cmpKey := componentKey(string(component.Component), component.Namespace)
		cmpList[cmpKey] = false
	}

	return &ComponentsStatus{components: cmpList}
}

func (cs *ComponentsStatus) ConsumeEvent(event deployment.ProcessUpdate) {
	// check if installation process failed
	if event.Event == deployment.ProcessExecutionFailure ||
		event.Event == deployment.ProcessForceQuitFailure ||
		event.Event == deployment.ProcessTimeoutFailure {
		cs.installationFailed = fmt.Errorf("status: %s, error: %v", event.Event, event.Error)
		return
	}

	// check if component installation process succeeded
	key := componentKey(event.Component.Name, event.Component.Namespace)
	if _, ok := cs.components[key]; !ok {
		cs.installationFailed = fmt.Errorf("component %s not exist for current runtime", key)
		return
	}

	if event.Component.Status == components.StatusInstalled {
		cs.components[key] = true
		return
	}

	cs.componentFailed = fmt.Errorf("component %s failed, status: %s", key, event.Component.Status)
}

func (cs *ComponentsStatus) StatusDescription() string {
	var installed int
	for _, c := range cs.components {
		if c {
			installed++
		}
	}

	return fmt.Sprintf("%d of %d components installed", installed, len(cs.components))
}

func (cs *ComponentsStatus) IsFinished() bool {
	for _, c := range cs.components {
		if !c {
			return false
		}
	}
	return true
}

func (cs *ComponentsStatus) ComponentError() error {
	// component error does not mean installation process failed,
	// component installation will be repeated
	return cs.componentFailed
}

func (cs *ComponentsStatus) InstallationError() error {
	return cs.installationFailed
}

func componentKey(name, namespace string) string {
	return fmt.Sprintf("%s-%s", name, namespace)
}
