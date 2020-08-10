package runtime

import "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

// GenericComponentDisabler provides functionality for removing configured component from given list
type GenericComponentDisabler struct {
	componentName      string
}

// NewOptionalComponent returns new instance of GenericComponentDisabler
func NewGenericComponentDisabler(name string) *GenericComponentDisabler {
	return &GenericComponentDisabler{componentName: name}
}

// Disable removes component form given lists. Filtering without allocating.
//
// source: https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
func (g *GenericComponentDisabler) Disable(components internal.ComponentConfigurationInputList) internal.ComponentConfigurationInputList {
	filterOut := components[:0]
	for _, component := range components {
		if !g.shouldRemove(component.Component) {
			filterOut = append(filterOut, component)
		}
	}

	for i := len(filterOut); i < len(components); i++ {
		components[i] = nil
	}

	return filterOut
}

func (g *GenericComponentDisabler) shouldRemove(in string) bool {
	return in == g.componentName
}
