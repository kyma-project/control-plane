package migrations

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type InstanceDetailsMigration struct {
	operations storage.Operations
	log        logrus.FieldLogger
}

func NewInstanceDetailsMigration(operations storage.Operations, log logrus.FieldLogger) *InstanceDetailsMigration {
	return &InstanceDetailsMigration{
		operations: operations,
		log:        log,
	}
}

func (m *InstanceDetailsMigration) Migrate() error {
	upgradeOperations, err := m.operations.ListUpgradeKymaOperations()
	if err != nil {
		return errors.Wrap(err, "while listing operations")
	}
	m.log.Infof("Performing instance details migration of %d operations", len(upgradeOperations))

	for _, op := range upgradeOperations {
		m.log.Infof("Existing upgradeKyma operation: %+v", op)
		if op.InstanceDetails.RuntimeID != "" {
			m.log.Infof("InstanceDetails were found in operation %s, skipping", op.Operation.ID)
			continue
		}
		lastProvOp, err := m.operations.GetProvisioningOperationByInstanceID(op.InstanceID)
		if err != nil {
			return errors.Wrap(err, "while listing operations")
		}
		op.InstanceDetails = lastProvOp.InstanceDetails
		//_, err = m.operations.UpdateUpgradeKymaOperation(op)
		//if err != nil {
		//	return errors.Wrap(err, "while updating operation parameters")
		//}
		m.log.Infof("Operation %s was migrated", op.Operation.ID)
		m.log.Infof("New upgradeKyma operation: %+v", op)
	}

	m.log.Info("Instance details migration end up successfully")
	return nil
}
