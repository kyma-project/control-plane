package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlansSchemaValidatorErrors(t *testing.T) {
	tests := map[string]struct {
		againstPlans []string
		inputJSON    string
		expErr       string
	}{
		"missing name": {
			againstPlans: []string{TrialPlanID},
			inputJSON:    `{"region": "munich"}`,
			expErr:       `(root): name is required`,
		},
		"not valid name": {
			againstPlans: []string{AzurePlanID},
			inputJSON:    `{"name": "wrong name"}`,
			expErr:       `name: Does not match pattern '^[a-zA-Z0-9-]*$'`,
		},
		"not valid machine type": {
			againstPlans: []string{AzurePlanID},
			inputJSON:    `{"name": "wrong-machType", "machineType": "WrongName"}`,
			expErr:       `machineType: machineType must be one of the following: "Standard_D8_v3"`,
		},
		"missing name, not valid region": {
			againstPlans: []string{AzurePlanID},
			inputJSON:    `{"region": "munich"}`,
			expErr:       `(root): name is required, region: region must be one of the following: "eastus", "centralus", "westus2", "uksouth", "northeurope", "westeurope", "japaneast", "southeastasia"`,
		},
	}
	for tN, tC := range tests {
		t.Run(tN, func(t *testing.T) {
			// given
			validator, err := NewPlansSchemaValidator(PlansConfig{})
			require.NoError(t, err)

			for _, id := range tC.againstPlans {
				// when
				result, err := validator[id].ValidateString(tC.inputJSON)
				require.NoError(t, err)

				// then
				assert.False(t, result.Valid)
				assert.EqualError(t, result.Error, tC.expErr)
			}
		})
	}
}

func TestNewPlansSchemaValidatorSuccess(t *testing.T) {
	// given
	validJSON := `{"name": "only-name-is-required"}`

	validator, err := NewPlansSchemaValidator(PlansConfig{})
	require.NoError(t, err)

	for _, id := range []string{GCPPlanID, AzurePlanID, AzureHAPlanID, TrialPlanID} {
		// when
		result, err := validator[id].ValidateString(validJSON)
		require.NoError(t, err)

		// then
		assert.True(t, result.Valid)

		// Currently there is a "bug" in /kyma-project/control-plane/components/director/pkg/jsonschema/validator.go:84
		// which missing executing method `.ErrorOrNil()` so we cannot use `assert.NoError`
		assert.Nil(t, result.Error)
	}
}
