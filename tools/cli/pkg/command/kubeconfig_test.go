package command

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsEuAccessTrueForSubAccountRegionCFEU11(t *testing.T) {
	subAccountRegion := "cf-eu11"

	isEuAcessResult := isEuAccess(subAccountRegion)

	require.Equal(t, isEuAcessResult, true)
}

func TestIsEuAccessTrueForSubAccountRegionCFCH20(t *testing.T) {
	subAccountRegion := "cf-ch20"

	isEuAcessResult := isEuAccess(subAccountRegion)

	require.Equal(t, isEuAcessResult, true)
}

func TestIsEuAccessFalseForANonEUSubAccountRegion(t *testing.T) {
	subAccountRegion := "cf-us10"

	isEuAcessResult := isEuAccess(subAccountRegion)

	require.Equal(t, isEuAcessResult, false)
}
