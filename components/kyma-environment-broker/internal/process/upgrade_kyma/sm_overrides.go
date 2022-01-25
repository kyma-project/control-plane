package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const (
	ServiceManagerComponentName       = "service-manager-proxy"
	HelmBrokerComponentName           = "helm-broker"
	ServiceCatalogComponentName       = "service-catalog"
	ServiceCatalogAddonsComponentName = "service-catalog-addons"
)

type ServiceManagerOverridesStep struct {
	operationManager *process.UpgradeKymaOperationManager
}

func NewServiceManagerOverridesStep(os storage.Operations) *ServiceManagerOverridesStep {
	return &ServiceManagerOverridesStep{
		operationManager: process.NewUpgradeKymaOperationManager(os),
	}
}

func (s *ServiceManagerOverridesStep) Name() string {
	return "ServiceManagerOverrides"
}

func (s *ServiceManagerOverridesStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	creds, err := operation.ProvideServiceManagerCredentials(log)
	if err != nil {
		log.Errorf("unable to obtain SM credentials: %s", err)
		return s.operationManager.OperationFailed(operation, err.Error(), log)
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
	operation.InputCreator.AppendOverrides(ServiceManagerComponentName, smOverrides)
	operation.InputCreator.DisableOptionalComponent(internal.BTPOperatorComponentName)

	operation.InputCreator.EnableOptionalComponent(HelmBrokerComponentName)
	operation.InputCreator.EnableOptionalComponent(ServiceCatalogComponentName)
	operation.InputCreator.EnableOptionalComponent(ServiceCatalogAddonsComponentName)
	operation.InputCreator.EnableOptionalComponent(ServiceManagerComponentName)

	return operation, 0, nil
}
