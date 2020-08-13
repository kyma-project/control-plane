package runtime

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type DisabledComponentsService struct {
	disabledComponents map[string]struct{}
}

// NewDisabledComponentsService returns new instance of ResourceSupervisorAggregator
func NewDisabledComponentsService(disabledComponents map[string]struct{}) *DisabledComponentsService {
	return &DisabledComponentsService{
		disabledComponents: disabledComponents,
	}
}

// DisableComponents executes disablers on given input and returns modified list.
//
// BE AWARE: in current implementation the input is also modified.
func (f *DisabledComponentsService) DisableComponents(components internal.ComponentConfigurationInputList) (internal.ComponentConfigurationInputList, error) {
	var filterOut = components
	for name := range f.disabledComponents {
		filterOut = NewGenericComponentDisabler(name).Disable(filterOut)
	}

	return filterOut, nil
}
