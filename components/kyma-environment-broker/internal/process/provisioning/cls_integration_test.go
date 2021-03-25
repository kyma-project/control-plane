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

	fakeGlobalAccountID := "fake-global-account-id"

	db := storage.NewMemoryStorage()
	fakeEncryptionKey := "1234567890123456"
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	provisioningManager := NewManager(db.Operations(), event.NewPubSub(log), log)
	provisioningSteps := []Step{
		NewClsOfferingStep(clsConfig, db.Operations()),
		NewClsProvisionStep(clsConfig, cls.NewProvisioner(db.CLSInstances(), clsClient), db.Operations()),
		NewClsCheckStatus(clsConfig, cls.NewStatusChecker(db.CLSInstances(), clsClient), db.Operations()),
		NewClsBindStep(clsConfig, clsClient, db.Operations(), fakeEncryptionKey),
	}
	for i, step := range provisioningSteps {
		provisioningManager.AddStep(i, step)
	}

	fakeOperationID1 := "fake-operation-id-1"
	fakeSKRInstanceID1 := "fake-skr-instance-id-1"
	operation1 := createProvisioningOperation(fakeOperationID1, fakeSKRInstanceID1, fakeGlobalAccountID, smClient)

	t.Log("Executing the CLS provisioning steps for the first time (no CLS exists for the global account)")
	err := db.Operations().InsertProvisioningOperation(operation1)
	require.NoError(t, err)
	_, err = provisioningManager.Execute(operation1.ID)
	require.NoError(t, err)

	foundCLS, exists, err := db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, foundCLS.References(), 1)
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID1))

	foundOp, err := db.Operations().GetProvisioningOperationByID(fakeOperationID1)
	require.NoError(t, err)
	require.NotEmpty(t, foundOp.Cls.BindingID)
	require.NotEmpty(t, foundOp.Cls.Overrides)
	require.True(t, foundOp.Cls.Instance.Provisioned)

	fakeOperationID2 := "fake-operation-id-2"
	fakeSKRInstanceID2 := "fake-skr-instance-id-2"
	operation2 := createProvisioningOperation(fakeOperationID2, fakeSKRInstanceID2, fakeGlobalAccountID, smClient)
	err = db.Operations().InsertProvisioningOperation(operation2)
	require.NoError(t, err)

	t.Log("Executing the CLS provisioning steps for the second time (there is an existing CLS for the global account)")
	_, err = provisioningManager.Execute(operation2.ID)
	require.NoError(t, err)

	foundCLS, exists, err = db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, foundCLS.References(), 2)
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID1))
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID2))

	foundOp, err = db.Operations().GetProvisioningOperationByID(fakeOperationID1)
	require.NoError(t, err)
	require.NotEmpty(t, foundOp.Cls.BindingID)
	require.NotEmpty(t, foundOp.Cls.Overrides)
	require.True(t, foundOp.Cls.Instance.Provisioned)
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

func createProvisioningOperation(operationID, skrInstanceID, globalAccountID string, smClient servicemanager.Client) internal.ProvisioningOperation {
	operation := fixture.FixProvisioningOperation(operationID, skrInstanceID)
	operation.Cls = internal.ClsData{}
	operation.SMClientFactory = servicemanager.NewPassthroughServiceManagerClientFactory(smClient)
	operation.State = domain.InProgress
	operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
	return operation
}

func createMockServiceManagerClient() servicemanager.Client {
	fakeClsOfferingID := "fake-cls-offering-id"
	fakeClsServiceID := "fake-cls-service-id"
	fakeClsBrokerID := "fake-cls-broker-id"
	fakeClsPlanID := "fake-cls-plan-id"

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
