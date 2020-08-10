package runtime_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/stretchr/testify/assert"
)

func TestGenericComponentDisabler(t *testing.T) {
	type toDisable struct {
		Name string
	}
	tests := []struct {
		name            string
		givenComponents []*string
		expComponents   []*string
		toDisable       toDisable
	}{
		{
			name: "Disable component if the name and namespace match with predicate",
			toDisable: toDisable{
				Name: "ory",
			},
			givenComponents: []*string{
				ptrStr("dex"),
				ptrStr("ory"),
			},
			expComponents: []*string{
				ptrStr("dex"),
			},
		},
		{
			name: "Disable component if name does not match",
			toDisable: toDisable{
				Name: "not-valid",
			},
			givenComponents: []*string{
				ptrStr("dex"),
				ptrStr("ory"),
			},
			expComponents: []*string{
				ptrStr("dex"),
				ptrStr("ory"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			sut := runtime.NewOptionalComponent(test.toDisable.Name)

			// when
			modifiedComponents := sut.Disable(test.givenComponents)

			// then
			assert.EqualValues(t, test.expComponents, modifiedComponents)
		})
	}
}

func ptrStr(s string) *string {
	return &s
}
