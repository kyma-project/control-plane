package migrations

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type InstanceParametersMigrationRollback struct {
	instances storage.Instances
	log       logrus.FieldLogger
}

func NewInstanceParametersMigrationRollback(instances storage.Instances, log logrus.FieldLogger) *InstanceParametersMigrationRollback {
	return &InstanceParametersMigrationRollback{
		instances: instances,
		log:       log,
	}
}

func (m *InstanceParametersMigrationRollback) Migrate() error {
	instances, _, _, err := m.instances.List(dbmodel.InstanceFilter{})
	if err != nil {
		return errors.Wrap(err, "while listing instances")
	}
	m.log.Infof("Performing instance parameters migration rollback of %d instances", len(instances))
	for _, i := range instances {
		m.log.Infof("updating instance %s without encryption", i.InstanceID)
		_, err = m.instances.UpdateWithoutEncryption(i)
		if err != nil {
			return errors.Wrap(err, "while updating instance")
		}
	}
	m.log.Info("Instance parameters migration end up successfully")
	return nil
}
