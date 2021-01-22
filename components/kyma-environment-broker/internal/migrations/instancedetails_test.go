package migrations_test

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
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
	return internal.ProvisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Lms: internal.LMS{
					TenantID: "lms-tenant",
				},
				SubAccountID: "subacc-id",
				RuntimeID:    "runtime-id",
				ShootName:    "shoot-name",
				ShootDomain:  "shoot-domain",
				XSUAA: internal.XSUAAData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:              "broker-id",
						ServiceID:             "service-id",
						PlanID:                "plan-id",
						InstanceID:            "instance-id",
						Provisioned:           true,
						ProvisioningTriggered: true,
					},
					XSAppname: "xsapp-name",
					BindingID: "binding-id",
				},
			},
			ID:         "prov-op-id",
			Version:    0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now().Add(1 * time.Minute),
			InstanceID: "instance-id",
			State:      domain.Succeeded,
		},
	}
}
