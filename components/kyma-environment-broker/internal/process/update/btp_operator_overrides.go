package update

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/sirupsen/logrus"
)

const BTPOperatorComponentName = "btp-operator"

type BTPOperatorOverridesStep struct {
	components input.ComponentListProvider
}

func NewBTPOperatorOverridesStep(components input.ComponentListProvider) *BTPOperatorOverridesStep {
	return &BTPOperatorOverridesStep{
		components: components,
	}
}

func (s *BTPOperatorOverridesStep) Name() string {
	return "BTPOperatorOverrides"
}

func (s *BTPOperatorOverridesStep) Run(operation internal.UpdatingOperation, logger logrus.FieldLogger) (internal.UpdatingOperation, time.Duration, error) {
	add := true
	for _, c := range operation.LastRuntimeState.ClusterSetup.KymaConfig.Components {
		if c.Component == BTPOperatorComponentName {
			add = false
		}
	}
	if add {
		c, err := getComponentInput(s.components, BTPOperatorComponentName, operation.RuntimeVersion)
		if err != nil {
			return operation, 0, err
		}
		setBTPOperatorOverrides(&c, operation)
		operation.LastRuntimeState.ClusterSetup.KymaConfig.Components = append(operation.LastRuntimeState.ClusterSetup.KymaConfig.Components, c)
	}
	return operation, 0, nil
}

func setBTPOperatorOverrides(c *reconciler.Components, operation internal.UpdatingOperation) {
	sm := operation.ProvisioningParameters.ErsContext.ServiceManager
	creds := sm.BTPOperatorCredentials
	c.Configuration = []reconciler.Configuration{
		{
			Key:    "manager.secret.clientid",
			Value:  creds.ClientID,
			Secret: true,
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  creds.ClientSecret,
			Secret: true,
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
			//TODO: this won't be part of the payload
			Key:   "cluster.id",
			Value: creds.ClusterID,
		},
	}
}
