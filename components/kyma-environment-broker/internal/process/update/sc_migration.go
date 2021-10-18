package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/sirupsen/logrus"
)

const (
	SCMigrationComponentName = "sc-migration"
)

type SCMigrationStep struct {
	components input.ComponentListProvider
}

func NewSCMigrationStep(components input.ComponentListProvider) *SCMigrationStep {
	return &SCMigrationStep{
		components: components,
	}
}

func (s *SCMigrationStep) Name() string {
	return "SCMigration"
}

func (s *SCMigrationStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	add := true
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component == SCMigrationComponentName {
			add = false
		}
	}
	if add {
		c, err := getComponentInput(s.components, SCMigrationComponentName, operation.RuntimeVersion)
		if err != nil {
			return operation, 0, err
		}
		operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = append(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components, c)
	}
	return operation, 0, nil
}
