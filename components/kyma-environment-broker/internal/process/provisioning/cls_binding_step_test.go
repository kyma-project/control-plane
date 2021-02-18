package provisioning

import (
	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	clsMock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClsEncryptDecrypt(t *testing.T) {
	// given
	secretKey := "1234567890123456"
	overridesIn := cls.ClsOverrides{
		FluentdEndPoint: "foo.bar",
		FluentdPassword: "fooPass",
		FluentdUsername: "fooUser",
		KibanaUrl:       "Kiib.url",
	}

	// when
	encrypted, err := encryptClsOverrides(secretKey, &overridesIn)
	assert.NoError(t, err)
	overridesOut, err := decryptClsOverrides(secretKey, encrypted)
	assert.NoError(t, err)

	// then
	assert.Equal(t, overridesIn, *overridesOut)
}
func TestClsBindingStep_Run(t *testing.T) {
	//given
	fakeRegion:= "westeurope"
	db := storage.NewMemoryStorage()
	repo := db.Operations()
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	clientFactory.SynchronousProvisioning()

	inputCreatorMock := &automock.ProvisionerInputCreator{}
	defer inputCreatorMock.AssertExpectations(t)
	expectedOverride := `    [OUTPUT]
        Name              http
        Match             *
        Host              fooEndPoint
        Port              443
        HTTP_User         fooUser
        HTTP_Passwd       fooPass
        tls               true
        tls.verify        true
        tls.debug         1
        URI               /
        Format            json`
	expectedKibanaUrl:= "kibUrl"
	inputCreatorMock.On("AppendOverrides", "logging", []*gqlschema.ConfigEntryInput{
		{
			Key:   "fluent-bit.config.outputs.additional",
			Value: expectedOverride,
		},
	}).Return(nil).Once()

	inputCreatorMock.On("SetLabel", kibanaURLLabelKey, expectedKibanaUrl).Return(nil).Once()

	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: internal.ProvisioningParametersDTO{Region: &fakeRegion},
				ErsContext: internal.ERSContext{SubAccountID: "1234567890", GlobalAccountID: "123-456-789"}},

			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{Instance: internal.ServiceManagerInstanceInfo{
					BrokerID:  fakeBrokerID,
					ServiceID: "svc-id",
					PlanID:    "plan-id",
				}},
				ShootDomain: "cls-test.sap.com",
			},
		},
		SMClientFactory: clientFactory,
		InputCreator: inputCreatorMock,
	}
	operation.Cls.Instance.ProvisioningTriggered = true
	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

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
	clsBindingProvider := &clsMock.ClsBindingProvider{}
	clsBindingProvider.On("CreateBinding", mock.Anything, mock.Anything).Return(&cls.ClsOverrides{
		FluentdEndPoint: "fooEndPoint",
		FluentdPassword: "fooPass",
		FluentdUsername: "fooUser",
		KibanaUrl:       "kibUrl",
	}, nil)

	bindingStep := NewClsBindStep(config, clsBindingProvider, repo, "1234567890123456")



	repo.InsertProvisioningOperation(operation)
	log := logger.NewLogDummy()
	// when
	operation, retry, err := bindingStep.Run(operation, log)
	require.NoError(t, err)
	require.Zero(t, retry)

}