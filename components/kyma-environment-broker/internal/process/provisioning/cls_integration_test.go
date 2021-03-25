package provisioning

import (
	"testing"

	smtypes "github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClsSteps(t *testing.T) {
	clsConfig := createDummyConfig()
	clsClient := cls.NewClient(clsConfig)
	smClient := createMockServiceManagerClient()

	operation := fixture.FixProvisioningOperation("operation-id", "instance-id")
	operation.Cls = internal.ClsData{}
	operation.SMClientFactory = servicemanager.NewPassthroughServiceManagerClientFactory(smClient)
	operation.State = domain.InProgress

	db := storage.NewMemoryStorage()
	encryptionKey := "1234567890123456"
	err := db.Operations().InsertProvisioningOperation(operation)
	require.NoError(t, err)

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	provisioningManager := NewManager(db.Operations(), event.NewPubSub(log), log)

	provisioningSteps := []Step{
		NewClsOfferingStep(clsConfig, db.Operations()),
		NewClsProvisionStep(clsConfig, cls.NewProvisioner(db.CLSInstances(), clsClient), db.Operations()),
		NewClsBindStep(clsConfig, clsClient, db.Operations(), encryptionKey),
	}

	for i, step := range provisioningSteps {
		provisioningManager.AddStep(i, step)
	}

	_, err = provisioningManager.Execute(operation.ID)
	require.NoError(t, err)
}

func createDummyConfig() *cls.Config {
	return &cls.Config{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		ServiceManager: &cls.ServiceManagerConfig{
			Credentials: []*cls.ServiceManagerCredentials{
				{
					Region:   "eu",
					URL:      "SM_URL",
					Username: "SM_USERNAME",
					Password: "SM_PASSWORD",
				},
			},
		},
		SAML: &cls.SAMLConfig{
			AdminGroup:  "runtimeAdmin",
			ExchangeKey: "SAML_EXCHANGE_KEY",
			RolesKey:    "groups",
			Idp: &cls.SAMLIdpConfig{
				EntityID:    "https://kymatest.accounts400.ondemand.com",
				MetadataURL: "https://kymatest.accounts400.ondemand.com/saml2/metadata",
			},
			Sp: &cls.SAMLSpConfig{
				EntityID:            "cls-dev",
				SignaturePrivateKey: "SAML_SIGNATURE_PRIVATE_KEY",
			},
		},
	}
}

func createMockServiceManagerClient() servicemanager.Client {
	fakeClsOfferingID := "cls-offering-id"
	fakeClsServiceID := "cls-service-id"
	fakeClsBrokerID := "cls-broker-id"
	fakeClsPlanID := "cls-plan-id"

	clientMock := &smautomock.Client{}
	clientMock.On("ListOfferingsByName", "cloud-logging").
		Return(&smtypes.ServiceOfferings{
			ServiceOfferings: []smtypes.ServiceOffering{{ID: fakeClsOfferingID, CatalogID: fakeClsServiceID, BrokerID: fakeClsBrokerID}},
		}, nil)
	clientMock.On("ListPlansByName", "standard", fakeClsOfferingID).
		Return(&smtypes.ServicePlans{
			ServicePlans: []smtypes.ServicePlan{{ID: fakeClsPlanID}},
		}, nil)
	clientMock.On("Provision", fakeClsBrokerID, mock.Anything, true).
		Return(&servicemanager.ProvisionResponse{}, nil)
	clientMock.On("LastInstanceOperation", mock.Anything, "").
		Return(servicemanager.LastOperationResponse{
			State: servicemanager.Succeeded,
		}, nil)
	clientMock.On("Bind", mock.Anything, mock.Anything, mock.Anything, false).
		Return(&servicemanager.BindingResponse{
			Binding: servicemanager.Binding{
				Credentials: map[string]interface{}{
					"Fluentd-username": "fluentd",
					"Fluentd-password": "fluentd",
					"Fluentd-endpoint": "fluentd.com",
					"Kibana-endpoint":  "kibana.com",
				},
			},
		}, nil)

	return clientMock
}
