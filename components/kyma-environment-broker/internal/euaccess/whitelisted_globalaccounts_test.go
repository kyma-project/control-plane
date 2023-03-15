package euaccess

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWhitelistedGlobalAccountIdsFromFile(t *testing.T) {
	// given/when
	d, err := ReadWhitelistedGlobalAccountIdsFromFile("test/eu_access_whitelist.yaml")

	// then
	require.NoError(t, err)
	assert.Equal(t, 2, len(d))
	assert.Equal(t, struct{}{}, d["whitelisted-global-account-id"])
	assert.Equal(t, struct{}{}, d["another-whitelisted-global-account-id"])
}
