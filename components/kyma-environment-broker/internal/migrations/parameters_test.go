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
	for tn, tc := range map[string]struct {
		operations []internal.LegacyOperation
		isError    bool
	}{
		"none": {
			operations: []internal.LegacyOperation{},
			isError:    true,
		},
		"single": {
			operations: []internal.LegacyOperation{
				fixLegacyOperation("test-1"),
			},
			isError: false,
		},
		"many": {
			operations: []internal.LegacyOperation{
				fixLegacyOperation("test-1"),
				fixLegacyOperation("test-2"),
			},
			isError: false,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			s := storage.NewMemoryStorage()
			log := logrus.New()

			for _, op := range tc.operations {
				err := s.Operations().InsertLegacyOperation(op)
				require.NoError(t, err)
			}

			err := migrations.NewParametersMigration(s.Operations(), log).Migrate()
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				for _, op := range tc.operations {
					operation, err := s.Operations().GetLegacyOperation(op.ID)
					require.NoError(t, err)

					pp := internal.ProvisioningParameters{}
					err = json.Unmarshal([]byte(fixProvisioningParameters()), &pp)
					require.NoError(t, err)

					assert.Equal(t, pp, operation.Operation.ProvisioningParameters)
				}
			}
		})
	}
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
