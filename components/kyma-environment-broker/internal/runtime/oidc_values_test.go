package runtime

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFilePath = "testdata/oidc-values.yaml"

func TestReadOIDCDefaultValuesFromYAML(t *testing.T) {

	t.Run("should read default OIDC values", func(t *testing.T) {
		// given
		expectedOidcValues := internal.OIDCConfigDTO{
			ClientID:    "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
			GroupsClaim: "groups",
			IssuerURL:   "https://kymatest.accounts400.ondemand.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		// when
		oidcValues, err := ReadOIDCDefaultValuesFromYAML(testFilePath)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedOidcValues, oidcValues)
	})

	t.Run("should return error while reading YAML file", func(t *testing.T) {
		// given
		nonExistentFilePath := "not/existent/file.yaml"

		// when
		oidcValues, err := ReadOIDCDefaultValuesFromYAML(nonExistentFilePath)

		// then
		require.Error(t, err)
		assert.Equal(t, internal.OIDCConfigDTO{}, oidcValues)
	})
}
