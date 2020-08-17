package runtime_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/stretchr/testify/assert"
)

func TestGenericComponentDisabler(t *testing.T) {
	type toDisable struct {
		Name string
	}
	tests := []struct {
		name            string
		givenComponents internal.ComponentConfigurationInputList
		expComponents   internal.ComponentConfigurationInputList
		toDisable       toDisable
	}{
		{
			name: "Disable component if the name and namespace match with predicate",
			toDisable: toDisable{
				Name: "ory",
			},
			givenComponents: internal.ComponentConfigurationInputList{
				{Component: "dex"},
				{Component: "ory"},
			},
			expComponents: internal.ComponentConfigurationInputList{
				{Component: "dex"},
			},
		},
		{
			name: "Disable component if name does not match",
			toDisable: toDisable{
				Name: "not-valid",
			},
			givenComponents: internal.ComponentConfigurationInputList{
				{Component: "dex"},
				{Component: "ory"},
			},
			expComponents: internal.ComponentConfigurationInputList{
				{Component: "dex"},
				{Component: "ory"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			sut := runtime.NewGenericComponentDisabler(test.toDisable.Name)

			// when
			modifiedComponents := sut.Disable(test.givenComponents)

			// then
			assert.EqualValues(t, test.expComponents, modifiedComponents)
		})
	}
}
