package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BusolaMigratorComponentName = "busola-migrator"

type BusolaMigratorOverridesStep struct{}

func NewBusolaMigratorOverridesStep() *BusolaMigratorOverridesStep {
	return &BusolaMigratorOverridesStep{}
}

func (s *BusolaMigratorOverridesStep) Name() string {
	return "BusolaMigratorOverrides"
}

func (s *BusolaMigratorOverridesStep) Run(operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	KubeconfigIDOverride := []*gqlschema.ConfigEntryInput{
		{
			Key:   "deployment.env.instanceID",
			Value: operation.InstanceID,
		},
		{
			Key:   "global.istio.gateway.name",
			Value: "kyma-gateway",
		},
	}

	operation.InputCreator.AppendOverrides(BusolaMigratorComponentName, KubeconfigIDOverride)
	return operation, 0, nil
}
