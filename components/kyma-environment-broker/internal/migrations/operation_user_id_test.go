package migrations_test

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOperationsUserIDMigration_Migrate(t *testing.T) {
	t.Run("should remove userID from existing DeprovisioningOperation", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		log := logrus.New()

		// given
		operation := fixDeprovisioningOperation()
		err := s.Deprovisioning().InsertDeprovisioningOperation(operation)
		require.NoError(t, err)

		// when
		err = migrations.NewOperationsUserIDMigration(s.Operations(), log).Migrate()
		require.NoError(t, err)

		// then
		op, err := s.Operations().GetDeprovisioningOperationByID(operation.ID)
		require.NoError(t, err)
		assert.Equal(t, "", op.ProvisioningParameters.ErsContext.UserID)
	})
}

func fixDeprovisioningOperation() internal.DeprovisioningOperation {
	return fixture.FixDeprovisioningOperation("test", "instance-id")
}
