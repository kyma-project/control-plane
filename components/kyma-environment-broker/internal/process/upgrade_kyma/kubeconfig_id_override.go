package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BusolaMigratorComponentName = "busola-migrator"

type BusolaMigratorOverridesStep struct{}

func NewBusolaMigratorOverridesStep(os storage.Operations) *BusolaMigratorOverridesStep {
	return &BusolaMigratorOverridesStep{}
}

func (s *BusolaMigratorOverridesStep) Name() string {
	return "InstanceIDOverride"
}

func (s *BusolaMigratorOverridesStep) Run(operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	KubeconfigIDOverride := []*gqlschema.ConfigEntryInput{
		{
			Key:   "deployment.env.instanceID",
			Value: operation.InstanceID,
		},
	}

	operation.InputCreator.AppendOverrides(BusolaMigratorComponentName, KubeconfigIDOverride)
	return operation, 0, nil
}
