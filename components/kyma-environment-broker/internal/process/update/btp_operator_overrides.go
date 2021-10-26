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
	creds := operation.ProvisioningParameters.ErsContext.SMOperatorCredentials
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
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: creds.URL,
		},
		{
			// TODO: get this from
			// https://github.com/kyma-project/kyma/blob/dba460de8273659cd8cd431d2737015a1d1909e5/tests/fast-integration/skr-svcat-migration-test/test-helpers.js#L39-L42
			Key:   "cluster.id",
			Value: "",
		},
	}
}
