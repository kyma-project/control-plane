package provisioning

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

func (s *BTPOperatorOverridesStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	sm := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
	overrides := []*gqlschema.ConfigEntryInput{
		{
			Key:    "manager.secret.clientid",
			Value:  sm.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  sm.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "manager.secret.url",
			Value: sm.ServiceManagerURL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: sm.URL,
		},
		{
			Key:   "cluster.id",
			Value: operation.InstanceDetails.RuntimeID, //
		},
	}
	operation.InputCreator.AppendOverrides(BTPOperatorComponentName, overrides)
	operation.InputCreator.EnableOptionalComponent(BTPOperatorComponentName)

	return operation, 0, nil
}
