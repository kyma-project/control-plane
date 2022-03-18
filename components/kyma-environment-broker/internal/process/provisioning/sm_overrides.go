package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

type ServiceManagerOverridesStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewServiceManagerOverridesStep(os storage.Operations) *ServiceManagerOverridesStep {
	return &ServiceManagerOverridesStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

func (s *ServiceManagerOverridesStep) Name() string {
	return "ServiceManagerOverrides"
}

func (s *ServiceManagerOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	creds, err := operation.ProvideServiceManagerCredentials(log)
	if err != nil {
		log.Errorf("unable to obtain SM credentials: %s", err)
		return s.operationManager.OperationFailed(operation, "unable to obtain SM credentials", err, log)
	}

	smOverrides := []*gqlschema.ConfigEntryInput{
		{
			Key:   "config.sm.url",
			Value: creds.URL,
		},
		{
			Key:   "sm.user",
			Value: creds.Username,
		},
		{
			Key:    "sm.password",
			Value:  creds.Password,
			Secret: ptr.Bool(true),
		},
	}
	operation.InputCreator.AppendOverrides(internal.ServiceManagerComponentName, smOverrides)

	operation.InputCreator.EnableOptionalComponent(internal.HelmBrokerComponentName)
	operation.InputCreator.EnableOptionalComponent(internal.ServiceCatalogComponentName)
	operation.InputCreator.EnableOptionalComponent(internal.ServiceCatalogAddonsComponentName)

	operation.InputCreator.EnableOptionalComponent(internal.ServiceManagerComponentName)

	return operation, 0, nil
}
