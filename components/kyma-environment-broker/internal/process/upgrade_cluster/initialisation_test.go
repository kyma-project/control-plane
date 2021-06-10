package upgrade_cluster

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma/automock"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
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

func createEvalManagerWithValidity(t *testing.T, storage storage.BrokerStorage, log *logrus.Logger, valid bool) (*avs.EvaluationManager, *avs.Client) {
	server := avs.NewMockAvsServer(t)
	mockServer := avs.FixMockAvsServer(server)
	client, err := avs.NewClient(context.TODO(), avs.Config{
		OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
		ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
	}, logrus.New())
	require.NoError(t, err)

	if !valid {
		client, err = avs.NewClient(context.TODO(), avs.Config{}, logrus.New())
	}
	require.NoError(t, err)

	avsDel := avs.NewDelegator(client, avs.Config{}, storage.Operations())
	upgradeEvalManager := avs.NewEvaluationManager(avsDel, avs.Config{})

	return upgradeEvalManager, client
}

func createEvalManager(t *testing.T, storage storage.BrokerStorage, log *logrus.Logger) (*avs.EvaluationManager, *avs.Client) {
	return createEvalManagerWithValidity(t, storage, log, true)
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

		upgradeOperation := fixUpgradeClusterOperation()
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient,
			nil, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)

	})

	t.Run("should initialize UpgradeRuntimeInput request when run", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, _ := createEvalManager(t, memoryStorage, log)

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		upgradeOperation := fixUpgradeClusterOperation()
		upgradeOperation.ProvisionerOperationID = ""
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		inputBuilder := &automock.CreatorForPlan{}
		inputBuilder.On("CreateUpgradeShootInput", fixProvisioningParameters()).Return(&input.RuntimeInput{}, nil)

		expectedOperation := upgradeOperation
		expectedOperation.Version++
		expectedOperation.State = orchestration.InProgress

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		op, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		inputBuilder.AssertNumberOfCalls(t, "CreateUpgradeShootInput", 1)
		assert.Equal(t, time.Duration(0), repeat)
		assert.NotNil(t, op.InputCreator)

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(op.Operation.ID)
		op.InputCreator = nil
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

		upgradeOperation := fixUpgradeClusterOperation()
		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), nil, nil, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, orchestration.Canceled, string(upgradeOperation.State))

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusActive, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusActive, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NotNil(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Failed, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
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
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
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

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManager, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: "", Original: ""})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

	t.Run("should retry on client error (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		_, client := createEvalManager(t, memoryStorage, log)
		evalManagerInvalid, _ := createEvalManagerWithValidity(t, memoryStorage, log, false)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := avs.StatusInactive, avs.StatusActive
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		provisionerClient := &provisionerAutomock.Client{}
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(
			gqlschema.OperationStatus{
				ID:        ptr.String(fixProvisionerOperationID),
				Operation: "",
				State:     gqlschema.OperationStateSucceeded,
				Message:   nil,
				RuntimeID: StringPtr(fixRuntimeID),
			}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManagerInvalid, nil)

		// when
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, repeat)
		assert.Equal(t, domain.InProgress, upgradeOperation.State)
		assert.Equal(t, internal.AvsEvaluationStatus{Current: internalStatus, Original: internalStatus}, upgradeOperation.Avs.AvsInternalEvaluationStatus)
		assert.Equal(t, internal.AvsEvaluationStatus{Current: externalStatus, Original: ""}, upgradeOperation.Avs.AvsExternalEvaluationStatus)
	})

	t.Run("should go through init and finish steps (both monitors)", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		evalManager, client := createEvalManager(t, memoryStorage, log)
		evalManagerInvalid, _ := createEvalManagerWithValidity(t, memoryStorage, log, false)
		inputBuilder := &automock.CreatorForPlan{}

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.InProgress})
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		internalStatus, externalStatus := avs.StatusInactive, avs.StatusActive
		avsData := createMonitors(t, client, internalStatus, externalStatus)
		upgradeOperation := fixUpgradeClusterOperationWithAvs(avsData)

		err = memoryStorage.Operations().InsertUpgradeClusterOperation(upgradeOperation)
		require.NoError(t, err)

		instance := fixInstanceRuntimeStatus()
		err = memoryStorage.Instances().Insert(instance)
		require.NoError(t, err)

		callCounter := 0
		provisionerClient := &provisionerAutomock.Client{}
		// for the first 2 step.Run calls, RuntimeOperationStatus will return OperationStateInProgress
		// otherwise, OperationStateSucceeded
		provisionerClient.On("RuntimeOperationStatus", fixGlobalAccountID, fixProvisionerOperationID).Return(
			func(accountID string, operationID string) gqlschema.OperationStatus {
				callCounter++
				if callCounter <= 2 {
					return gqlschema.OperationStatus{
						ID:        ptr.String(fixProvisionerOperationID),
						Operation: "",
						State:     gqlschema.OperationStateInProgress,
						Message:   nil,
						RuntimeID: StringPtr(fixRuntimeID),
					}
				}

				return gqlschema.OperationStatus{
					ID:        ptr.String(fixProvisionerOperationID),
					Operation: "",
					State:     gqlschema.OperationStateSucceeded,
					Message:   nil,
					RuntimeID: StringPtr(fixRuntimeID),
				}
			}, nil)

		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Orchestrations(), provisionerClient, inputBuilder, evalManagerInvalid, nil)

		// when invalid client request, this should be delayed
		upgradeOperation, repeat, err := step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, repeat)
		assert.Equal(t, domain.InProgress, upgradeOperation.State)
		assert.Equal(t, internal.AvsEvaluationStatus{Current: internalStatus, Original: internalStatus}, upgradeOperation.Avs.AvsInternalEvaluationStatus)
		assert.Equal(t, internal.AvsEvaluationStatus{Current: externalStatus, Original: ""}, upgradeOperation.Avs.AvsExternalEvaluationStatus)

		// when valid client request and InProgress state from RuntimeOperationStatus, this should do init tasks
		step.evaluationManager = evalManager
		upgradeOperation, repeat, err = step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, 1*time.Minute, repeat)
		assert.Equal(t, domain.InProgress, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusMaintenance, Original: internalStatus})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: avs.StatusMaintenance, Original: externalStatus})

		// when valid client request and Succeeded state from RuntimeOperationStatus, this should do finish tasks
		upgradeOperation, repeat, err = step.Run(upgradeOperation, log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, domain.Succeeded, upgradeOperation.State)
		assert.Equal(t, upgradeOperation.Avs.AvsInternalEvaluationStatus, internal.AvsEvaluationStatus{Current: internalStatus, Original: avs.StatusMaintenance})
		assert.Equal(t, upgradeOperation.Avs.AvsExternalEvaluationStatus, internal.AvsEvaluationStatus{Current: externalStatus, Original: avs.StatusMaintenance})

		storedOp, err := memoryStorage.Operations().GetUpgradeClusterOperationByID(upgradeOperation.Operation.ID)
		assert.Equal(t, upgradeOperation, *storedOp)
		assert.NoError(t, err)
	})

}

func fixUpgradeClusterOperation() internal.UpgradeClusterOperation {
	return fixUpgradeClusterOperationWithAvs(internal.AvsLifecycleData{})
}

func fixUpgradeClusterOperationWithAvs(avsData internal.AvsLifecycleData) internal.UpgradeClusterOperation {
	upgradeOperation := fixture.FixUpgradeClusterOperation(fixUpgradeOperationID, fixInstanceID)
	upgradeOperation.OrchestrationID = fixOrchestrationID
	upgradeOperation.ProvisionerOperationID = fixProvisionerOperationID
	upgradeOperation.State = orchestration.Pending
	upgradeOperation.Description = ""
	upgradeOperation.UpdatedAt = time.Now()
	upgradeOperation.InstanceDetails.Avs = avsData
	upgradeOperation.ProvisioningParameters = fixProvisioningParameters()
	upgradeOperation.RuntimeOperation.GlobalAccountID = fixGlobalAccountID
	upgradeOperation.RuntimeOperation.SubAccountID = fixSubAccountID
	upgradeOperation.InputCreator = nil

	return upgradeOperation
}

func fixProvisioningOperation() internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(fixProvisioningOperationID, fixInstanceID)
	provisioningOperation.ProvisionerOperationID = fixProvisionerOperationID
	provisioningOperation.Description = ""
	provisioningOperation.ProvisioningParameters = fixProvisioningParameters()

	return provisioningOperation
}

func fixProvisioningParameters() internal.ProvisioningParameters {
	pp := fixture.FixProvisioningParameters("1")
	pp.PlanID = broker.AzurePlanID
	pp.ServiceID = ""
	pp.ErsContext.GlobalAccountID = fixGlobalAccountID
	pp.ErsContext.SubAccountID = fixSubAccountID

	return pp
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(fixInstanceID)
	instance.RuntimeID = fixRuntimeID
	instance.GlobalAccountID = fixGlobalAccountID

	return instance
}

func StringPtr(s string) *string {
	return &s
}
