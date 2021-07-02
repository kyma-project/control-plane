package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BusolaMigratorComponentName = "busola-migrator"

type BusolaMigratorOverridesStep struct {
}

func NewBusolaMigratorOverridesStep(os storage.Operations) *BusolaMigratorOverridesStep {
	return &BusolaMigratorOverridesStep{}
}

func (s *BusolaMigratorOverridesStep) Name() string {
	return "InstanceIDOverride"
}

func (s *BusolaMigratorOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	kubeconfigIDOverrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "deployment.env.kubeconfigID",
			Value: operation.InstanceID,
		},
	}

	operation.InputCreator.AppendOverrides(BusolaMigratorComponentName, kubeconfigIDOverrides)
	return operation, 0, nil
}
