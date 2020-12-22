package migrations

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ParametersMigration struct {
	operations storage.Operations
	log        logrus.FieldLogger
}

func NewParametersMigration(operations storage.Operations, log logrus.FieldLogger) *ParametersMigration {
	return &ParametersMigration{
		operations: operations,
		log:        log,
	}
}

func (p *ParametersMigration) Migrate() error {
	operations, size, _, err := p.operations.ListOperations(dbmodel.OperationFilter{})
	if err != nil {
		return errors.Wrap(err, "while listing operations")
	}

	opsParams, err := p.operations.ListOperationsParameters()
	if err != nil {
		return errors.Wrap(err, "while listing operations parameters")
	}

	p.log.Infof("Performing parameters migration of %d operations", size)

	for _, op := range operations {
		if op.ProvisioningParameters.PlanID != "" {
			p.log.Infof("Provisioning parameters were found in operation %s, skipping", op.ID)
			continue
		}
		oldProvisioningParameters, ok := opsParams[op.ID]
		if !ok {
			p.log.Infof("Old provisioning parameters for operation %s were not found, skipping", op.ID)
			continue
		}
		op.ProvisioningParameters = oldProvisioningParameters
		_, err := p.operations.UpdateOperationParameters(op)
		if err != nil {
			return errors.Wrap(err, "while updating operation parameters")
		}
		p.log.Infof("Operation %s was migrated", op.ID)
	}

	p.log.Info("Provisioning parameters migration end up successfully")
	return nil
}
