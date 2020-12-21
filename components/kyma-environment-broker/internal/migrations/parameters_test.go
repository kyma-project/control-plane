package migrations_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParametersMigration_Migrate(t *testing.T) {
	s := storage.NewMemoryStorage()
	log := logrus.New()

	operationID := "test"

	err := s.Operations().InsertLegacyOperation(fixLegacyOperation(operationID))
	require.NoError(t, err)

	err = migrations.NewParametersMigration(s.Operations(), log).Migrate()
	require.NoError(t, err)

	operation, err := s.Operations().GetOperationByID(operationID)
	require.NoError(t, err)

	pp := internal.ProvisioningParameters{}
	err = json.Unmarshal([]byte(fixProvisioningParameters()), &pp)
	require.NoError(t, err)

	assert.Equal(t, pp, operation.ProvisioningParameters)
}

func fixLegacyOperation(id string) internal.LegacyOperation {
	return internal.LegacyOperation{
		Operation: internal.Operation{
			ID:                     id,
			InstanceID:             id,
			ProvisionerOperationID: id,
			UpdatedAt:              time.Now(),
		},
		ProvisioningParameters: fixProvisioningParameters(),
	}
}

func fixProvisioningParameters() string {
	return `{
			"plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
			"ers_context": {
				"subaccount_id": "b9b1"
			},
			"parameters": {
				"name": "test",
				"region": "westeurope"
			}
		}`
}
