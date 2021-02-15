package migrations

import (
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
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
	instances, _, _, err := m.instances.List(dbmodel.InstanceFilter{})
	if err != nil {
		return errors.Wrap(err, "while listing instances")
	}
	m.log.Infof("Performing instance parameters migration of %d instances", len(instances))
	for _, i := range instances {
		_, err = m.cipher.Decrypt([]byte(i.Parameters.ErsContext.ServiceManager.Credentials.BasicAuth.Username))
		switch {
		case err == nil:
			m.log.Infof("instance %s was already migrated", i.InstanceID)
			return nil
		// not valid format errors in this scenario means that input was not encrypted - we perform encryption for them
		case kebError.IsNotValidFormatError(err):
			m.log.Infof("updating instance %s", i.InstanceID)
			_, err = m.instances.Update(i)
			if err != nil {
				return errors.Wrap(err, "while updating instance")
			}
		case err != nil:
			return errors.Wrap(err, "while decrypting username")
		}
	}
	m.log.Info("Instance parameters migration end up successfully")
	return nil
}
