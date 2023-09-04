package avs_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maintenanceModeGAIDsYamlFilePath = "testdata/avs_maintenance_mode_always_disabled_ga_ids.yaml"
)

func TestConfig_ReadMaintenanceModeDuringUpgradeAlwaysDisabledGAIDsFromYaml(t *testing.T) {
	// given
	avsConfig := avs.Config{}
	expectedGAID1, expectedGAID2 := "test-ga-id-1", "test-ga-id-2"

	// when
	err := avsConfig.ReadMaintenanceModeDuringUpgradeAlwaysDisabledGAIDsFromYaml(maintenanceModeGAIDsYamlFilePath)
	require.NoError(t, err)

	// then
	assert.Contains(t, avsConfig.MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs, expectedGAID1)
	assert.Contains(t, avsConfig.MaintenanceModeDuringUpgradeAlwaysDisabledGAIDs, expectedGAID2)
}
