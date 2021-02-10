package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"

	"testing"
)

const (
	fakeBrokerID = "fake-broker-id"
)


func TestClsProvisioningStep_Run(t *testing.T) {
	fakeRegion := "fooRegion"

	//given
	db := storage.NewMemoryStorage()
	repo := db.Operations()
	// TODO: Change this to new servicemanager instatiation
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	clientFactory.SynchronousProvisioning()
	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{
				Parameters: internal.ProvisioningParametersDTO{Region: &fakeRegion},
				ErsContext: internal.ERSContext{SubAccountID: "1234567890"}},

			InstanceDetails: internal.InstanceDetails{
				Cls: internal.ClsData{Instance: internal.ServiceManagerInstanceInfo{
					BrokerID:  "broker-id",
					ServiceID: "svc-id",
					PlanID:    "plan-id",
				}},
				ShootDomain: "cls-test.sap.com",
			},
		},
		SMClientFactory: clientFactory,

	}

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
					Region:   "foo",
					URL:      "https://foo.bar",
					Username: "fooUser",
					Password: "barPassword",
				},
			},
		},
	}
	clsClient := cls.NewClient(config, logs.WithField("service", "clsClient"))
	clsInstanceManager := cls.NewInstanceManager(db.CLSInstances(), clsClient, logs.WithField("service", "clsInstanceManager"))

	skrRegion := operation.ProvisioningParameters.Parameters.Region
	smClientMock := &automock.Client{}
	clsInstanceManagerMock := &authorizationRuleN
	smClientMock.On("ServiceManagerClient", operation.SMClientFactory, config.ServiceManager, skrRegion).Return(nil, nil)
	smClientMock.On("CreateInstance", smClientMock, &cls.CreateInstanceRequest{BrokerID: fakeBrokerID}).Return(fakeBrokerID, nil)

	offeringStep := NewClsOfferingStep(config,repo)

	provisionStep := NewClsProvisioningStep(config, clsInstanceManager, repo)
	repo.InsertProvisioningOperation(operation)

	log := logger.NewLogDummy()
	// when
	operation, retry, err := offeringStep.Run(operation, log)
	require.NoError(t, err)
	require.Zero(t, retry)

	operation, retry, err = provisionStep.Run(operation, logger.NewLogDummy())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	assert.NotEmpty(t, operation.Cls.Instance.InstanceID)
	assert.False(t, operation.Cls.Instance.Provisioned)
	assert.True(t, operation.Cls.Instance.ProvisioningTriggered)
	clientFactory.AssertProvisionCalled(t, servicemanager.InstanceKey{
		BrokerID:   "broker-id",
		InstanceID: operation.Cls.Instance.InstanceID,
		ServiceID:  "svc-id",
		PlanID:     "plan-id",
	})
}