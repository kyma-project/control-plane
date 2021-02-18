package migrations

import (
	"reflect"

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
		logger := m.log.WithField("UpgradeKymaOperation", op.Operation.ID)
		logger.Infof("Found existing upgradeKyma operation %s", op.Operation.ID)

		lastProvOp, err := m.operations.GetProvisioningOperationByInstanceID(op.InstanceID)
		if err != nil {
			return errors.Wrap(err, "while getting operations")
		}
		if reflect.DeepEqual(op.InstanceDetails, lastProvOp.InstanceDetails) {
			m.log.Infof("InstanceDetails were found in operation %s, skipping", op.Operation.ID)
			continue
		}
		logger.Infof("Last provisioningOperation %s", lastProvOp.Operation.ID)
		if lastProvOp.InstanceDetails.RuntimeID == "" {
			m.log.Warnf("Empty InstanceDetails for provisioningOperation: %s", lastProvOp.Operation.ID)
		}

		op.InstanceDetails = lastProvOp.InstanceDetails
		_, err = m.operations.UpdateUpgradeKymaOperation(op)
		if err != nil {
			return errors.Wrap(err, "while updating operation parameters")
		}
		logger.Infof("Operation %s was migrated", op.Operation.ID)
	}

	m.log.Info("Instance details migration end up successfully")
	return nil
}
