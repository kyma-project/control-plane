package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"

	"testing"

	"github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	clsMock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
)

const (
	fakeBrokerID = "fake-broker-id"
)

func TestClsProvisioningStep_Run(t *testing.T) {
	fakeRegion := "westeurope"

	//given
	db := storage.NewMemoryStorage()
	repo := db.Operations()
	clientFactory := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{}, []types.ServicePlan{})
	clientFactory.SynchronousProvisioning()
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
					Region:   "eu",
					URL:      "https://foo.bar",
					Username: "fooUser",
					Password: "barPassword",
				},
			},
		},
	}

	provisionerMock := &clsMock.ClsProvisioner{}
	provisionerMock.On("Provision", mock.Anything, mock.Anything, &cls.ProvisionRequest{
		GlobalAccountID: operation.ProvisioningParameters.ErsContext.GlobalAccountID,
		Region:          "eu",
		Instance: servicemanager.InstanceKey{
			BrokerID:  fakeBrokerID,
			ServiceID: "svc-id",
			PlanID:    "plan-id",
		},
	}).Return(&cls.ProvisionResult{
		InstanceID: "instance_id",
	}, nil)

	offeringStep := NewClsOfferingStep(config, repo)

	provisionStep := NewClsProvisionStep(config, provisionerMock, repo)
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
}
