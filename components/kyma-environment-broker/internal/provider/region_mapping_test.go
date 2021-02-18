package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPlatformRegionMappingFromFile(t *testing.T) {
	// given/when
	d, err := ReadPlatformRegionMappingFromFile("test/regions.yaml")

	// then
	require.NoError(t, err)
	assert.Equal(t, "europe", d["cf-eu"])
	assert.Equal(t, "us", d["cf-us"])
}
