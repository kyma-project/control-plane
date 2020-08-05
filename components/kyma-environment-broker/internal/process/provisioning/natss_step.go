package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const (
	NatsStreamingStepName = "NatsStreaming"
)

type NatsStreamingStep struct {
	operationManager *process.ProvisionOperationManager
}

// ensure the interface is implemented
var _ Step = (*NatsStreamingStep)(nil)

func NewNatsStreamingStep(os storage.Operations) *NatsStreamingStep {
	return &NatsStreamingStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *NatsStreamingStep) Name() string {
	return NatsStreamingStepName
}

func (s *NatsStreamingStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	parameters, err := operation.GetProvisioningParameters()
	if err != nil {
		log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
	}
	log.Infof(NatsStreamingStepName+"Provisioning parameters from operation: %v", parameters)

	// TODO finish the implementation
	//
	// append installation overrides
	operation.InputCreator.AppendOverrides("nats-streaming", getNatsStreamingOverrides())

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