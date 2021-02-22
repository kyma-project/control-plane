package migrations

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/driver/postsql"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type InstanceParametersMigration struct {
	instances storage.Instances
	cipher    postsql.Cipher
	log       logrus.FieldLogger
}

func NewInstanceParametersMigration(instances storage.Instances, cipher postsql.Cipher, log logrus.FieldLogger) *InstanceParametersMigration {
	return &InstanceParametersMigration{
		instances: instances,
		cipher:    cipher,
		log:       log,
	}
}

func (m *InstanceParametersMigration) Migrate() error {
	instances, _, _, err := m.instances.ListWithoutDecryption(dbmodel.InstanceFilter{})
	if err != nil {
		return errors.Wrap(err, "while listing instances")
	}
	m.log.Infof("Performing instance parameters migration of %d instances", len(instances))
	for _, i := range instances {
		if i.Parameters.ErsContext.ServiceManager != nil {
			_, err = m.cipher.Decrypt([]byte(i.Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Username))
			if err == nil {
				m.log.Infof("instance %s was already migrated", i.InstanceID)
			} else {
				m.log.Infof("updating instance %s", i.InstanceID)
				_, err = m.instances.Update(i)
				if err != nil {
					return errors.Wrap(err, "while updating instance")
				}
			}
		}
	}
	m.log.Info("Instance parameters migration end up successfully")
	return nil
}
