package config_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	additionalComponentsConfigKey = "additional-components"
)

func TestValidate(t *testing.T) {
	// setup
	cfgValidator := config.NewConfigMapKeysValidator()

	t.Run("should validate whether config contains required fields", func(t *testing.T) {
		// given
		cfgString := `additional-components:
  - name: "additional-component1"
    namespace: "kyma-system"
optional-field: "optional"`

		// when
		err := cfgValidator.Validate(cfgString)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error indicating missing required fields", func(t *testing.T) {
		// given
		cfgString := `optional-field: "optional"`

		// when
		err := cfgValidator.Validate(cfgString)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), additionalComponentsConfigKey)
	})
}
