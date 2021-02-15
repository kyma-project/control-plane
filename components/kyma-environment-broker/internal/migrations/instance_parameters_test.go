package migrations_test

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestInstanceParametersMigration_Migrate(t *testing.T) {
	t.Run("should encrypt Instance parameters", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		log := logrus.New()

		// given
		testValue := "test"
		err := s.Instances().Insert(internal.Instance{
			InstanceID: testValue,
			Parameters: internal.ProvisioningParameters{
				ErsContext: internal.ERSContext{
					ServiceManager: &internal.ServiceManagerEntryDTO{
						Credentials: internal.ServiceManagerCredentials{
							BasicAuth: internal.ServiceManagerBasicAuth{
								Username: testValue,
								Password: testValue,
							},
						},
					},
				},
			},
		})
		require.NoError(t, err)

		err = migrations.NewInstanceDetailsMigration(s.Operations(), log).Migrate()
		require.NoError(t, err)
	})
}
