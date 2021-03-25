package provisioning

import (
	"context"
	"testing"
	"time"

	smtypes "github.com/Peripli/service-manager-cli/pkg/types"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type finishProvisioningStep struct {
	operationManager *process.ProvisionOperationManager
}

func newFinishProvisioningStep(repo storage.Operations) *finishProvisioningStep {
	return &finishProvisioningStep{
		operationManager: process.NewProvisionOperationManager(repo),
	}
}

func (s *finishProvisioningStep) Name() string {
	return "CLS_Provision_Success"
}

func (s *finishProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	return s.operationManager.OperationSucceeded(operation, "", log)
}

type finishDeprovisioningStep struct {
	operationManager *process.DeprovisionOperationManager
}

func newFinishDeprovisioningStep(repo storage.Operations) *finishDeprovisioningStep {
	return &finishDeprovisioningStep{
		operationManager: process.NewDeprovisionOperationManager(repo),
	}
}

func (s *finishDeprovisioningStep) Name() string {
	return "CLS_Deprovision_Success"
}

func (s *finishDeprovisioningStep) Run(operation internal.DeprovisioningOperation, log logrus.FieldLogger) (internal.DeprovisioningOperation, time.Duration, error) {
	return s.operationManager.OperationSucceeded(operation, "", log)
}

func TestClsStepsWithFakeServiceManager(t *testing.T) {
	clsConfig := createDummyConfig()
	db := storage.NewMemoryStorage()
	smClientFactory := servicemanager.NewPassthroughServiceManagerClientFactory(createMockServiceManagerClient())

	runClsEndToEndFlow(t, clsConfig, db, smClientFactory)
}

func runClsEndToEndFlow(t *testing.T, clsConfig *cls.Config, db storage.BrokerStorage, smClientFactory internal.SMClientFactory) {
	ctx := context.TODO()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	clsClient := cls.NewClient(clsConfig)

	fakeGlobalAccountID := "fake-global-account-id"
	fakeEncryptionKey := "1234567890123456"

	provisioningManager := NewManager(db.Operations(), event.NewPubSub(log), log)
	provisioningSteps := []Step{
		NewClsOfferingStep(clsConfig, db.Operations()),
		NewClsProvisionStep(clsConfig, cls.NewProvisioner(db.CLSInstances(), clsClient), db.Operations()),
		NewClsCheckStatus(clsConfig, cls.NewStatusChecker(db.CLSInstances()), db.Operations()),
		NewClsBindStep(clsConfig, clsClient, db.Operations(), fakeEncryptionKey),
		newFinishProvisioningStep(db.Operations()),
	}
	for i, step := range provisioningSteps {
		provisioningManager.AddStep(i, step)
	}
	provisioningQueue := process.NewQueue(provisioningManager, log)
	provisioningQueue.Run(ctx.Done(), 1)

	deprovisioningManager := deprovisioning.NewManager(db.Operations(), event.NewPubSub(log), log)
	deprovisioningSteps := []deprovisioning.Step{
		deprovisioning.NewClsUnbindStep(clsConfig, db.Operations()),
		deprovisioning.NewClsDeprovisionStep(clsConfig, cls.NewDeprovisioner(db.CLSInstances(), clsClient), db.Operations()),
		newFinishDeprovisioningStep(db.Operations()),
	}
	for i, step := range deprovisioningSteps {
		deprovisioningManager.AddStep(i, step)
	}
	deprovisioningQueue := process.NewQueue(deprovisioningManager, log)
	deprovisioningQueue.Run(ctx.Done(), 1)

	log.Info("Executing the CLS provisioning steps for the first time (no CLS exists for the global account)")
	fakeOperationID1 := "fake-operation-id-1"
	fakeSKRInstanceID1 := "fake-skr-instance-id-1"
	op1 := createProvisioningOperation(fakeOperationID1, fakeSKRInstanceID1, fakeGlobalAccountID, smClientFactory)
	err := db.Operations().InsertProvisioningOperation(op1)
	require.NoError(t, err)
	provisioningQueue.Add(op1.ID)
	waitUntilSucceeds(op1.ID, db)

	foundCLS, exists, err := db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, foundCLS.References(), 1)
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID1))

	foundOp1, err := db.Operations().GetProvisioningOperationByID(fakeOperationID1)
	require.NoError(t, err)
	require.NotEmpty(t, foundOp1.Cls.BindingID)
	require.NotEmpty(t, foundOp1.Cls.Overrides)
	require.True(t, foundOp1.Cls.Instance.Provisioned)

	log.Info("Executing the CLS provisioning steps for the second time (there is an existing CLS for the global account)")
	fakeOperationID2 := "fake-operation-id-2"
	fakeSKRInstanceID2 := "fake-skr-instance-id-2"
	op2 := createProvisioningOperation(fakeOperationID2, fakeSKRInstanceID2, fakeGlobalAccountID, smClientFactory)
	err = db.Operations().InsertProvisioningOperation(op2)
	require.NoError(t, err)
	provisioningQueue.Add(op2.ID)
	waitUntilSucceeds(op2.ID, db)

	foundCLS, exists, err = db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, foundCLS.References(), 2)
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID1))
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID2))

	foundOp2, err := db.Operations().GetProvisioningOperationByID(fakeOperationID2)
	require.NoError(t, err)
	require.NotEmpty(t, foundOp2.Cls.BindingID)
	require.NotEmpty(t, foundOp2.Cls.Overrides)
	require.True(t, foundOp2.Cls.Instance.Provisioned)

	log.Info("Executing the CLS deprovisioning steps for the first time")
	fakeOperationID3 := "fake-operation-id-3"
	op3 := createDeprovisioningOperation(fakeOperationID3, foundOp1, smClientFactory)
	err = db.Operations().InsertDeprovisioningOperation(op3)
	require.NoError(t, err)
	deprovisioningQueue.Add(op3.ID)
	waitUntilSucceeds(op3.ID, db)

	foundCLS, exists, err = db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.True(t, exists)
	require.Len(t, foundCLS.References(), 1)
	require.True(t, foundCLS.IsReferencedBy(fakeSKRInstanceID2))

	log.Info("Executing the CLS deprovisioning steps for the second time")
	fakeOperationID4 := "fake-operation-id-4"
	op4 := createDeprovisioningOperation(fakeOperationID4, foundOp2, smClientFactory)
	err = db.Operations().InsertDeprovisioningOperation(op4)
	require.NoError(t, err)
	deprovisioningQueue.Add(op4.ID)
	waitUntilSucceeds(op4.ID, db)

	_, exists, err = db.CLSInstances().FindActiveByGlobalAccountID(fakeGlobalAccountID)
	require.NoError(t, err)
	require.False(t, exists)
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

func createProvisioningOperation(operationID, skrInstanceID, globalAccountID string, smClientFactory internal.SMClientFactory) internal.ProvisioningOperation {
	operation := fixture.FixProvisioningOperation(operationID, skrInstanceID)
	operation.Cls = internal.ClsData{}
	operation.State = domain.InProgress
	operation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
	operation.SMClientFactory = smClientFactory

	return operation
}

func createDeprovisioningOperation(operationID string, originalOp *internal.ProvisioningOperation, smClientFactory internal.SMClientFactory) internal.DeprovisioningOperation {
	operation := fixture.FixDeprovisioningOperation(operationID, originalOp.InstanceID)
	operation.Cls = originalOp.Cls
	operation.State = domain.InProgress
	operation.ProvisioningParameters.ErsContext.GlobalAccountID = originalOp.ProvisioningParameters.ErsContext.GlobalAccountID
	operation.SMClientFactory = smClientFactory

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
	clientMock.On("Unbind", mock.Anything, mock.Anything, true).
		Return(&servicemanager.DeprovisionResponse{}, nil)
	clientMock.On("Deprovision", mock.Anything, true).
		Return(&servicemanager.DeprovisionResponse{}, nil)

	return clientMock
}

func waitUntilSucceeds(opID string, db storage.BrokerStorage) {
	for {
		foundOp, _ := db.Operations().GetOperationByID(opID)
		if foundOp.State == domain.Failed {
			panic("Operation failed")
		}

		if foundOp.State == domain.Succeeded {
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}
