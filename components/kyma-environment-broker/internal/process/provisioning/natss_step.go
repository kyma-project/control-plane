package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

type NatsStreamingStep struct{}

// ensure the interface is implemented
var _ Step = (*NatsStreamingStep)(nil)

func NewNatsStreamingOverridesStep() *NatsStreamingStep {
	return &NatsStreamingStep{}
}

func (s *NatsStreamingStep) Name() string {
	return "Provision Nats Streaming"
}

func (s *NatsStreamingStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	log.Infof("Provisioning for PlanID: %s", operation.ProvisioningParameters.PlanID)
	operation.InputCreator.AppendOverrides(components.NatsStreaming, getNatsStreamingOverrides())
	return operation, 0, nil
}

func getNatsStreamingOverrides() []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:   "global.natsStreaming.persistence.enabled",
			Value: "false",
		},
	}
}
