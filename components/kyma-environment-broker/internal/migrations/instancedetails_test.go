package migrations_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestInstanceDetailsMigration_Migrate(t *testing.T) {
	t.Run("should migrate InstanceDetails from existing ProvisioningOperation", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		log := logrus.New()

		// given
		err := s.Provisioning().InsertProvisioningOperation(fixProvisioningOperation())
		require.NoError(t, err)
		err = s.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				InstanceDetails:        internal.InstanceDetails{},
				ID:                     "upgrade-op-id",
				CreatedAt:              time.Now().Add(5 * time.Minute),
				UpdatedAt:              time.Now().Add(6 * time.Minute),
				InstanceID:             "instance-id",
				State:                  orchestration.Canceled,
				ProvisioningParameters: internal.ProvisioningParameters{},
				OrchestrationID:        "orch-id",
			},
			RuntimeOperation: orchestration.RuntimeOperation{},
			InputCreator:     nil,
		})
		require.NoError(t, err)

		err = migrations.NewInstanceDetailsMigration(s.Operations(), log).Migrate()
		require.NoError(t, err)

	})
}

func fixProvisioningOperation() internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation("prov-op-id", "instance-id")
	provisioningOperation.Operation.InstanceDetails = fixture.FixInstanceDetails("id")
	provisioningOperation.Operation.InstanceDetails.XSUAA.Instance.Provisioned = true
	provisioningOperation.Operation.InstanceDetails.XSUAA.Instance.ProvisioningTriggered = true

	return provisioningOperation
}
