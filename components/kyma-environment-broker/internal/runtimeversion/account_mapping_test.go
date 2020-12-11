package runtimeversion

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	cmName             = "config"
	namespace          = "foo"
	versionForGA       = "1.14"
	versionForSA       = "1.15-rc1"
	fixGlobalAccountID = "628ee42b-bd1e-42b3-8a1d-c4726fd2ee62\n"
	fixSubAccountID    = "e083d3a8-5139-4705-959f-8279c86f6fe7\n"
)

func TestAccountVersionMapping_Get(t *testing.T) {
	t.Run("Should get version for SubAccount when both GlobalAccount and SubAccount are provided", func(t *testing.T) {
		// given
	svc := fixAccountVersionMapping(t, map[string]string{
		fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
		fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):    versionForSA,
	})

	// when
	version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
	require.NoError(t, err)

	// then
	assert.True(t, found)
	assert.Equal(t, versionForSA, version)
	assert.Equal(t, internal.SubAccount, origin)
	})

	t.Run("Should get version for GlobalAccount when only GlobalAccount is provided", func(t *testing.T) {
		// given
		svc := fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", globalAccountPrefix, fixGlobalAccountID): versionForGA,
		})

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.True(t, found)
		assert.Equal(t, versionForGA, version)
		assert.Equal(t, internal.GlobalAccount, origin)
	})

	t.Run("Should get version for SubAccount when only SubAccount is provided", func(t *testing.T) {
		// given
		svc := fixAccountVersionMapping(t, map[string]string{
			fmt.Sprintf("%s%s", subaccountPrefix, fixSubAccountID):    versionForSA,
		})

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.True(t, found)
		assert.Equal(t, versionForSA, version)
		assert.Equal(t, internal.SubAccount, origin)
	})

	t.Run("Should not get version when nothing is provided", func(t *testing.T) {
		// given
		svc := fixAccountVersionMapping(t, map[string]string{})

		// when
		version, origin, found, err := svc.Get(fixGlobalAccountID, fixSubAccountID)
		require.NoError(t, err)

		// then
		assert.False(t, found)
		assert.Empty(t, version)
		assert.Empty(t, origin)
	})
}
