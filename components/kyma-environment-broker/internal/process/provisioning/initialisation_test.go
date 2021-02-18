package provisioning

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/stretchr/testify/assert"
)

const (
	statusOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	statusInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	statusRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	statusGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	statusProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"

	dashboardURL               = "http://runtime.com"
	fixAvsEvaluationInternalId = int64(1234)
)

func TestInitialisationStep(t *testing.T) {
	t.Run("run initialized", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		operation := fixOperationRuntimeStatus(broker.GCPPlanID)
		err := memoryStorage.Operations().InsertProvisioningOperation(operation)
		assert.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", statusGlobalAccountID, statusProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(statusProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: ptr.String(operation.RuntimeID),
		}, nil)
		provisionerClient.On("RuntimeStatus", statusGlobalAccountID, operation.RuntimeID).Return(gqlschema.RuntimeStatus{
			LastOperationStatus:     nil,
			RuntimeConnectionStatus: nil,
			RuntimeConfiguration: &gqlschema.RuntimeConfig{ClusterConfig: &gqlschema.GardenerConfig{
				Name:   ptr.String("test-gardener-name"),
				Region: ptr.String("test-gardener-region"),
				Seed:   ptr.String("test-gardener-seed"),
			}},
		}, nil)

		directorClient := &automock.DirectorClient{}
		directorClient.On("GetConsoleURL", statusGlobalAccountID, statusRuntimeID).Return(dashboardURL, nil)

		mockOauthServer := newMockAvsOauthServer()
		defer mockOauthServer.Close()
		mockAvsSvc := newMockAvsService(t, false)
		mockAvsSvc.startServer()
		defer mockAvsSvc.server.Close()
		avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
		avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
		assert.NoError(t, err)
		avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
		externalEvalAssistant := avs.NewExternalEvalAssistant(avsConfig)
		externalEvalCreator := NewExternalEvalCreator(avsDel, false, externalEvalAssistant)
		internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
		InternalEvalUpdater := NewInternalEvalUpdater(avsDel, internalEvalAssistant, avsConfig)
		iasType := NewIASType(nil, true)

		rvc := &automock.RuntimeVersionConfiguratorForProvisioning{}
		defer rvc.AssertExpectations(t)

		// setup ProvisioningOperation and mockAvsService state to simulate InternalEvaluationStep execution
		operation.Avs.AvsEvaluationInternalId = fixAvsEvaluationInternalId
		mockAvsSvc.evals[fixAvsEvaluationInternalId] = fixAvsEvaluation()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient,
			directorClient, nil, externalEvalCreator, InternalEvalUpdater, iasType, time.Hour, time.Hour, rvc, nil)

		// when
		operation, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, operation.State)

		updatedInstance, err := memoryStorage.Instances().GetByID(statusInstanceID)
		assert.NoError(t, err)
		assert.Equal(t, dashboardURL, updatedInstance.DashboardURL)

		inDB, err := memoryStorage.Operations().GetProvisioningOperationByID(operation.ID)
		assert.NoError(t, err)
		assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AVSEvaluationExternalId)
		assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AvsEvaluationInternalId)
		assert.Equal(t, 4, len(mockAvsSvc.evals[inDB.Avs.AvsEvaluationInternalId].Tags))
	})

	t.Run("run unintialized", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		operation := fixOperationRuntimeStatus(broker.GCPPlanID)
		err := memoryStorage.Operations().InsertProvisioningOperation(operation)
		assert.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", statusGlobalAccountID, statusProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(statusProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: nil,
		}, nil)
		provisionerClient.On("RuntimeStatus", statusGlobalAccountID, operation.RuntimeID).Return(gqlschema.RuntimeStatus{
			LastOperationStatus:     nil,
			RuntimeConnectionStatus: nil,
			RuntimeConfiguration: &gqlschema.RuntimeConfig{ClusterConfig: &gqlschema.GardenerConfig{
				Name:   ptr.String("test-gardener-name"),
				Region: ptr.String("test-gardener-region"),
				Seed:   ptr.String("test-gardener-seed"),
			}},
		}, nil)

		directorClient := &automock.DirectorClient{}
		directorClient.On("GetConsoleURL", statusGlobalAccountID, statusRuntimeID).Return(dashboardURL, nil)

		mockOauthServer := newMockAvsOauthServer()
		defer mockOauthServer.Close()
		mockAvsSvc := newMockAvsService(t, false)
		mockAvsSvc.startServer()
		defer mockAvsSvc.server.Close()
		avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
		avsClient, err := avs.NewClient(context.TODO(), avsConfig, logger.NewLogDummy())
		assert.NoError(t, err)
		avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
		externalEvalAssistant := avs.NewExternalEvalAssistant(avsConfig)
		externalEvalCreator := NewExternalEvalCreator(avsDel, false, externalEvalAssistant)
		internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
		InternalEvalUpdater := NewInternalEvalUpdater(avsDel, internalEvalAssistant, avsConfig)
		iasType := NewIASType(nil, true)

		rvc := &automock.RuntimeVersionConfiguratorForProvisioning{}
		defer rvc.AssertExpectations(t)

		// setup ProvisioningOperation and mockAvsService state to simulate InternalEvaluationStep execution
		operation.Avs.AvsEvaluationInternalId = fixAvsEvaluationInternalId
		mockAvsSvc.evals[fixAvsEvaluationInternalId] = fixAvsEvaluation()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient,
			directorClient, nil, externalEvalCreator, InternalEvalUpdater, iasType, time.Hour, time.Hour, rvc, nil)

		// when
		operation, repeat, err := step.Run(operation, logger.NewLogDummy())

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, operation.State)

		updatedInstance, err := memoryStorage.Instances().GetByID(statusInstanceID)
		assert.NoError(t, err)
		assert.Equal(t, dashboardURL, updatedInstance.DashboardURL)

		inDB, err := memoryStorage.Operations().GetProvisioningOperationByID(operation.ID)
		assert.NoError(t, err)
		assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AVSEvaluationExternalId)
		assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AvsEvaluationInternalId)
		assert.Equal(t, 4, len(mockAvsSvc.evals[inDB.Avs.AvsEvaluationInternalId].Tags))
	})
}

func fixOperationRuntimeStatus(planId string) internal.ProvisioningOperation {
	provisioningOperation := internal.FixProvisioningOperation(statusOperationID, statusInstanceID)
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.InstanceDetails.RuntimeID = runtimeID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID

	return provisioningOperation
}

func fixOperationRuntimeStatusWithProvider(planId string, provider internal.TrialCloudProvider) internal.ProvisioningOperation {
	provisioningOperation := internal.FixProvisioningOperation(statusOperationID, statusInstanceID)
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.ProvisioningParameters.Parameters.Provider = &provider

	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := internal.FixInstance(statusInstanceID)
	instance.RuntimeID = statusRuntimeID
	instance.DashboardURL = dashboardURL
	instance.GlobalAccountID = statusGlobalAccountID

	return instance
}

func fixAvsEvaluation() *avs.BasicEvaluationCreateResponse {
	return &avs.BasicEvaluationCreateResponse{
		DefinitionType:   "av",
		Name:             "fake-internal-eval",
		Description:      "",
		Service:          "",
		URL:              "",
		CheckType:        "",
		Interval:         180,
		TesterAccessId:   int64(999),
		Timeout:          30000,
		ReadOnly:         false,
		ContentCheck:     "",
		ContentCheckType: "",
		Threshold:        30000,
		GroupId:          int64(4321),
		Visibility:       "PUBLIC",
		DateCreated:      time.Now().Unix(),
		DateChanged:      time.Now().Unix(),
		Owner:            "johndoe@xyz.corp",
		Status:           "ACTIVE",
		Alerts:           nil,
		Tags: []*avs.Tag{
			{
				Content:      "already-exist-tag",
				TagClassId:   123456,
				TagClassName: "already-exist-tag-classname",
			},
		},
		Id:                         fixAvsEvaluationInternalId,
		LegacyCheckId:              fixAvsEvaluationInternalId,
		InternalInterval:           60,
		AuthType:                   "AUTH_NONE",
		IndividualOutageEventsOnly: false,
		IdOnTester:                 "",
	}
}

func newInMemoryKymaVersionConfigurator(versions map[string]string) *inMemoryKymaVersionConfigurator {
	return &inMemoryKymaVersionConfigurator{
		perGAID: versions,
	}
}

type inMemoryKymaVersionConfigurator struct {
	perGAID map[string]string
}

func (c *inMemoryKymaVersionConfigurator) ForGlobalAccount(string) (string, bool, error) {
	return "", true, nil
}
