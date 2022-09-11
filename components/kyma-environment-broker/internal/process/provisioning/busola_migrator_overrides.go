package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BusolaMigratorComponentName = "busola-migrator"

type BusolaMigratorOverridesStep struct {
}

func NewBusolaMigratorOverridesStep() *BusolaMigratorOverridesStep {
	return &BusolaMigratorOverridesStep{}
}

func (s *BusolaMigratorOverridesStep) Name() string {
	return "BusolaMigratorOverrides"
}

func (s *BusolaMigratorOverridesStep) Run(operation internal.Operation, log logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	kubeconfigIDOverrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "deployment.env.kubeconfigID",
			Value: operation.InstanceID,
		},
		{
			Key:   "global.istio.gateway.name",
			Value: "kyma-gateway",
		},
	}

	operation.InputCreator.AppendOverrides(BusolaMigratorComponentName, kubeconfigIDOverrides)
	return operation, 0, nil
}
