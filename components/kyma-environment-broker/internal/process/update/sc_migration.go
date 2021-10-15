package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
)

type SCMigrationStep struct{}

func NewSCMigrationStep() *SCMigrationStep {
	return &SCMigrationStep{}
}

func (s *SCMigrationStep) Name() string {
	return "SCMigration"
}

func (s *SCMigrationStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	if ForMigration(operation) {
		operation.InputCreator.EnableOptionalComponent("sc-migration")
		operation.InputCreator.DisableOptionalComponent("service-catalog")
		operation.InputCreator.DisableOptionalComponent("service-catalog-addons")
		operation.InputCreator.DisableOptionalComponent("helm-broker")
	}
	return operation, 0, nil
}
