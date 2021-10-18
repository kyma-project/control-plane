package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

type BTPOperatorOverridesStep struct{}

func NewBTPOperatorOverridesStep() *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	sm := operation.ProvisioningParameters.ErsContext.ServiceManager
	creds := sm.BTPOperatorCredentials
	overrides := []*gqlschema.ConfigEntryInput{
		{
			Key:    "manager.secret.clientid",
			Value:  creds.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  creds.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "manager.secret.url",
			Value: sm.URL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: creds.TokenURL,
		},
		{
			Key:   "cluster.id",
			Value: creds.ClusterID,
		},
	}
	operation.InputCreator.AppendOverrides(BTPOperatorComponentName, overrides)
	operation.InputCreator.EnableOptionalComponent(BTPOperatorComponentName)
	operation.InputCreator.DisableOptionalComponent("service-manager-proxy")
	operation.InputCreator.DisableOptionalComponent("service-catalog")
	operation.InputCreator.DisableOptionalComponent("service-catalog-addons")
	operation.InputCreator.DisableOptionalComponent("helm-broker")
	return operation, 0, nil
}
