package migrations

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type OperationsUserIDMigration struct {
	operations storage.Operations
	log        logrus.FieldLogger
}

func NewOperationsUserIDMigration(operations storage.Operations, log logrus.FieldLogger) *OperationsUserIDMigration {
	return &OperationsUserIDMigration{
		operations: operations,
		log:        log,
	}
}

func (m *OperationsUserIDMigration) Migrate() error {
	operations, err := m.operations.ListDeprovisioningOperations()
	if err != nil {
		return errors.Wrap(err, "while listing operations")
	}
	m.log.Infof("Performing userID migration of %d operations", len(operations))

	for _, op := range operations {
		if op.ProvisioningParameters.ErsContext.UserID == "" || op.State != domain.Succeeded {
			m.log.Infof("Skipping migrating operation %s", op.ID)
			continue
		}
		m.log.Infof("Migrating operation %s", op.ID)
		op.ProvisioningParameters.ErsContext.UserID = ""
		_, err = m.operations.UpdateDeprovisioningOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating operation")
		}
	}

	m.log.Info("Operations userID migration end up successfully")
	return nil
}
