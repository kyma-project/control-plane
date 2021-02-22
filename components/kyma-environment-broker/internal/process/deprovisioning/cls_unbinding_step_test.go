package deprovisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"

	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestClsUnbindStep_Run(t *testing.T) {
	// given
	repo := storage.NewMemoryStorage().Operations()
	config := &cls.Config{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		SAML: &cls.SAMLConfig{
			AdminGroup:  "runtimeAdmin",
			ExchangeKey: "base64-jibber-jabber",
			RolesKey:    "groups",
			Idp: &cls.SAMLIdpConfig{
				EntityID:    "https://sso.example.org/idp",
				MetadataURL: "https://sso.example.org/idp/saml2/metadata",
			},
			Sp: &cls.SAMLSpConfig{
				EntityID:            "cls-dev",
				SignaturePrivateKey: "base64-jibber-jabber",
			},
		},
		ServiceManager: &cls.ServiceManagerConfig{
			Credentials: []*cls.ServiceManagerCredentials{
				{
					Region:   "eu",
					URL:      "https://foo.bar",
					Username: "fooUser",
					Password: "barPassword",
				},
			},
		},
	}
	step := NewClsUnbindStep(config, repo)
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})

	operation := internal.DeprovisioningOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{
					Instance: internal.ServiceManagerInstanceInfo{
						BrokerID:    "broker-id",
						ServiceID:   "svc-id",
						PlanID:      "plan-id",
						InstanceID:  "instance-id",
						Provisioned: true,
					},
					Binding: internal.BindingInfo{
						Bound:     true,
						BindingID: "binding-id",
					},
					Overrides: "clsOverrides",
				},
			},
		},
		SMClientFactory: clientFactory,
	}
	repo.InsertDeprovisioningOperation(operation)

	// when
	operation, retry, err := step.Run(operation, logger.NewLogDummy())

	// then
	require.NoError(t, err)
	assert.Zero(t, retry)
	assert.Empty(t, operation.Cls.Binding.BindingID)
	assert.False(t, operation.Cls.Binding.Bound)
	assert.Empty(t, operation.Cls.Overrides)
	clientFactory.AssertUnbindCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: "instance-id",
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	}, "binding-id")
}
