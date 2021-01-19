package upgrade_kyma

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	fixProvisioningOperationID = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	fixOrchestrationID         = "fd5cee4d-0eeb-40d0-a7a7-0708eseba470"
	fixUpgradeOperationID      = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	fixInstanceID              = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	fixRuntimeID               = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	fixGlobalAccountID         = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	fixSubAccountID            = "6424cc6d-5fce-49fc-b720-cf1fc1f36c7d"
	fixProvisionerOperationID  = "e04de524-53b3-4890-b05a-296be393e4ba"
)

func createMonitors(t *testing.T, client *avs.Client, internalStatus string, externalStatus string) internal.AvsLifecycleData {
	// monitors
	var (
		operationInternalId int64
		operationExternalId int64
	)

	// internal
	inMonitor, err := client.CreateEvaluation(&avs.BasicEvaluationCreateRequest{
		Name: "internal monitor",
	})
	require.NoError(t, err)
	operationInternalId = inMonitor.Id

	if avs.ValidStatus(internalStatus) {
		_, err = client.SetStatus(inMonitor.Id, internalStatus)
		require.NoError(t, err)
	}

	// external
	exMonitor, err := client.CreateEvaluation(&avs.BasicEvaluationCreateRequest{
		Name: "internal monitor",
	})
	require.NoError(t, err)
	operationExternalId = exMonitor.Id

	if avs.ValidStatus(externalStatus) {
		_, err = client.SetStatus(exMonitor.Id, externalStatus)
		require.NoError(t, err)
	}

	// return AvsLifecycleData
	avsData := internal.AvsLifecycleData{
		AvsEvaluationInternalId: operationInternalId,
		AVSEvaluationExternalId: operationExternalId,
		AvsInternalEvaluationStatus: internal.AvsEvaluationStatus{
			Current:  internalStatus,
			Original: "",
		},
		AvsExternalEvaluationStatus: internal.AvsEvaluationStatus{
			Current:  externalStatus,
			Original: "",
		},
		AVSInternalEvaluationDeleted: false,
		AVSExternalEvaluationDeleted: false,
	}

	return avsData
}

func createEvalManager(t *testing.T, storage storage.BrokerStorage, log *logrus.Logger) (*EvaluationManager, *avs.Client) {
	server := newServer(t)
	mockServer := fixHTTPServer(server)
	client, err := avs.NewClient(context.TODO(), avs.Config{
		OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
		ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
	}, logrus.New())
	require.NoError(t, err)
	avsDel := avs.NewDelegator(client, avs.Config{}, storage.Operations())
	upgradeEvalManager := NewEvaluationManager(avsDel, avs.Config{})

	return upgradeEvalManager, client
}

func TestInitialisationStep_Run(t *testing.T) {
	t.Run("should mark operation as Succeeded when upgrade was successful", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, _ := createEvalManager(t, memoryStorage, log)

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeKymaOperation()
		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, nil, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)

	})

	t.Run("should initialize UpgradeRuntimeInput request when run", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, _ := createEvalManager(t, memoryStorage, log)
		ver := &internal.RuntimeVersionData{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeKymaOperation()
		upgradeOperation.ProvisionerOperationID = ""
		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		inputBuilder := &automock.CreatorForPlan{}
		inputBuilder.On("CreateUpgradeInput", fixProvisioningParameters(), *ver).Return(&input.RuntimeInput{}, nil)

		rvc := &automock.RuntimeVersionConfiguratorForUpgrade{}
		defer rvc.AssertExpectations(t)
		expectedOperation := upgradeOperation
		expectedOperation.Version++
		expectedOperation.State = orchestration.InProgress
		rvc.On("ForUpgrade", expectedOperation).Return(ver, nil).Once()

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, rvc)

		// when
		op, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		inputBuilder.AssertNumberOfCalls(t, "CreateUpgradeInput", 1)
		assert.Equal(t, time.Duration(0), repeat)
		assert.NotNil(t, op.InputCreator)

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(op.Operation.ID)
		assert.Equal(t, op, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should mark finish if orchestration was canceled", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, _ := createEvalManager(t, memoryStorage, log)

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.Canceled})
		require.NoError(t, err)

		upgradeOperation := fixUpgradeKymaOperation()
		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), nil, nil, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, orchestration.Canceled, string(upgradeOperation.State))

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		require.NoError(t, err)
		assert.Equal(t, upgradeOperation, *storedOp)
	})

	t.Run("should refresh avs on success (both monitors, empty init)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		avsData := createMonitors(t, client, "", "")
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusActive, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusActive, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := avs.StatusActive, avs.StatusInactive
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on fail (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := avs.StatusActive, avs.StatusInactive
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateFailed,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NotNil(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Failed, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (internal monitor)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := avs.StatusActive, ""
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		avsData.AVSEvaluationExternalId = 0
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (external monitor)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := "", avs.StatusInactive
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		avsData.AvsEvaluationInternalId = 0
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should refresh avs on success (no monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := "", ""
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		avsData.AvsEvaluationInternalId = 0
		avsData.AVSEvaluationExternalId = 0
		upgradeOperation := fixUpgradeKymaOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(gqlschema.OperationStatus{
			ID:        ptr.String(fixProvisionerOperationID),
			Operation: "",
			State:     gqlschema.OperationStateSucceeded,
			Message:   nil,
			RuntimeID: StringPtr(fixRuntimeID),
		}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})

		storedOp, err := memoryStorage.Operations().GetUpgradeKymaOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

}

func fixUpgradeKymaOperation() internal.UpgradeKymaOperation {
	return fixUpgradeKymaOperationWithAvs(internal.AvsLifecycleData{})
}

func fixUpgradeKymaOperationWithAvs(avsData internal.AvsLifecycleData) internal.UpgradeKymaOperation {
	n := time.Now()
	windowEnd := n.Add(time.Minute)
	return internal.UpgradeKymaOperation{
		Operation: internal.Operation{
			InstanceDetails: internal.InstanceDetails{
				Avs: avsData,
			},
			ID:                     fixUpgradeOperationID,
			InstanceID:             fixInstanceID,
			OrchestrationID:        fixOrchestrationID,
			ProvisionerOperationID: fixProvisionerOperationID,
			State:                  orchestration.Pending,
			Description:            "",
			CreatedAt:              n,
			UpdatedAt:              n,
			ProvisioningParameters: fixProvisioningParameters(),
		},
		RuntimeOperation: orchestration.RuntimeOperation{
			Runtime: orchestration.Runtime{
				MaintenanceWindowEnd: windowEnd,
			},
		},
	}
}

func fixProvisioningOperation() internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: internal.Operation{
			ID:                     fixProvisioningOperationID,
			InstanceID:             fixInstanceID,
			ProvisionerOperationID: fixProvisionerOperationID,
			Description:            "",
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
			ProvisioningParameters: fixProvisioningParameters(),
		},
	}
}

func fixProvisioningParameters() internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:    broker.GCPPlanID,
		ServiceID: "",
		ErsContext: internal.ERSContext{
			GlobalAccountID: fixGlobalAccountID,
			SubAccountID:    fixSubAccountID,
		},
		Parameters: internal.ProvisioningParametersDTO{},
	}
}

func fixInstanceRuntimeStatus() internal.Instance {
	return internal.Instance{
		InstanceID:      fixInstanceID,
		RuntimeID:       fixRuntimeID,
		DashboardURL:    "",
		GlobalAccountID: fixGlobalAccountID,
		CreatedAt:       time.Time{},
		UpdatedAt:       time.Time{},
		DeletedAt:       time.Time{},
	}
}

func StringPtr(s string) *string {
	return &s
}

type evaluationRepository struct {
	basicEvals   map[int64]*avs.BasicEvaluationCreateResponse
	evalSet      map[int64]bool
	parentIDrefs map[int64][]int64
}

func (er *evaluationRepository) addEvaluation(parentID int64, eval *avs.BasicEvaluationCreateResponse) {
	er.basicEvals[eval.Id] = eval
	er.evalSet[eval.Id] = true
	er.parentIDrefs[parentID] = append(er.parentIDrefs[parentID], eval.Id)
}

func (er *evaluationRepository) removeParentRef(parentID, evalID int64) {
	refs := er.parentIDrefs[parentID]

	for i, evalWithRef := range refs {
		if evalID == evalWithRef {
			refs[i] = refs[len(refs)-1]
			er.parentIDrefs[parentID] = refs[:len(refs)-1]
		}
	}
}

const (
	parentEvaluationID     = 42
	evaluationName         = "test_evaluation"
	existingEvaluationName = "test-eval-name"
	accessToken            = "1234abcd"
	tokenType              = "test"
)

type server struct {
	t            *testing.T
	evaluations  *evaluationRepository
	tokenExpired int
}

func newServer(t *testing.T) *server {
	return &server{
		t: t,
		evaluations: &evaluationRepository{
			basicEvals:   make(map[int64]*avs.BasicEvaluationCreateResponse, 0),
			evalSet:      make(map[int64]bool, 0),
			parentIDrefs: make(map[int64][]int64, 0),
		},
	}
}

func fixHTTPServer(srv *server) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/oauth/token", srv.token).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata", srv.createEvaluation).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}", srv.deleteEvaluation).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}", srv.getEvaluation).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}/lifecycle", srv.setStatus).Methods(http.MethodPut)
	r.HandleFunc("/api/v2/evaluationmetadata/{parentId}/child/{evalId}", srv.removeReferenceFromParentEval).Methods(http.MethodDelete)

	return httptest.NewServer(r)
}
func (s *server) token(w http.ResponseWriter, _ *http.Request) {
	token := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	response, err := json.Marshal(token)
	assert.NoError(s.t, err)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(response)
	assert.NoError(s.t, err)

	w.WriteHeader(http.StatusOK)
}

func (s *server) hasAccess(token string) bool {
	if s.tokenExpired > 0 {
		s.tokenExpired--
		return false
	}
	if token == fmt.Sprintf("%s %s", tokenType, accessToken) {
		return true
	}

	return false
}

func (s *server) createEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj avs.BasicEvaluationCreateRequest
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.t, err)

	evalCreateResponse := s.createResponseObj(requestObj)
	s.evaluations.addEvaluation(requestObj.ParentId, evalCreateResponse)

	createdEval := s.evaluations.basicEvals[evalCreateResponse.Id]
	responseObjAsBytes, _ := json.Marshal(createdEval)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)

	w.WriteHeader(http.StatusOK)
}

func (s *server) getEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	response, exists := s.evaluations.basicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)
}

func (s *server) setStatus(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj string
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.t, err)

	if !avs.ValidStatus(requestObj) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	evaluation, exists := s.evaluations.basicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	evaluation.Status = requestObj

	responseObjAsBytes, _ := json.Marshal(evaluation)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)
}

func (s *server) deleteEvaluation(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	if _, exists := s.evaluations.basicEvals[id]; exists {
		delete(s.evaluations.basicEvals, id)
		delete(s.evaluations.evalSet, id)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *server) removeReferenceFromParentEval(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	parentID, err := strconv.ParseInt(vars["parentId"], 10, 64)
	assert.NoError(s.t, err)

	evalID, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	_, exists := s.evaluations.parentIDrefs[parentID]
	if !exists {
		w.WriteHeader(http.StatusBadRequest)
	}

	s.evaluations.removeParentRef(parentID, evalID)
}

func fixTag() *avs.Tag {
	return &avs.Tag{
		Content:    "test-tag",
		TagClassId: 111111,
	}
}

func (s *server) createResponseObj(requestObj avs.BasicEvaluationCreateRequest) *avs.BasicEvaluationCreateResponse {
	parsedThreshold, err := strconv.ParseInt(requestObj.Threshold, 10, 64)
	if err != nil {
		parsedThreshold = int64(1234)
	}

	timeUnixEpoch, id := s.generateId()

	evalCreateResponse := &avs.BasicEvaluationCreateResponse{
		DefinitionType:             requestObj.DefinitionType,
		Name:                       requestObj.Name,
		Description:                requestObj.Description,
		Service:                    requestObj.Service,
		URL:                        requestObj.URL,
		CheckType:                  requestObj.CheckType,
		Interval:                   requestObj.Interval,
		TesterAccessId:             requestObj.TesterAccessId,
		Timeout:                    requestObj.Timeout,
		ReadOnly:                   requestObj.ReadOnly,
		ContentCheck:               requestObj.ContentCheck,
		ContentCheckType:           requestObj.ContentCheck,
		Threshold:                  parsedThreshold,
		GroupId:                    requestObj.GroupId,
		Visibility:                 requestObj.Visibility,
		DateCreated:                timeUnixEpoch,
		DateChanged:                timeUnixEpoch,
		Owner:                      "abc@xyz.corp",
		Status:                     "ACTIVE",
		Alerts:                     nil,
		Tags:                       requestObj.Tags,
		Id:                         id,
		LegacyCheckId:              id,
		InternalInterval:           60,
		AuthType:                   "AUTH_NONE",
		IndividualOutageEventsOnly: false,
		IdOnTester:                 "",
	}
	return evalCreateResponse
}

func (s *server) generateId() (int64, int64) {
	for {
		timeUnixEpoch := time.Now().Unix()
		id := rand.Int63() + time.Now().Unix()
		if _, exists := s.evaluations.evalSet[id]; !exists {
			return timeUnixEpoch, id
		}
	}
}
