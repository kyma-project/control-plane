package upgrade_kyma

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"

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

func createEvalManager(storage storage.BrokerStorage, log *logrus.Logger) *EvaluationManager {
	ctx, _ := context.WithCancel(context.Background())
	avsClient, _ := avs.NewClient(ctx, avs.Config{}, log)
	avsDel := avs.NewDelegator(avsClient, avs.Config{}, storage.Operations())
	upgradeEvalManager := NewEvaluationManager(avsDel, avs.Config{})
	return upgradeEvalManager
}

func TestInitialisationStep_Run(t *testing.T) {
	t.Run("should mark operation as Succeeded when upgrade was successful", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()

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

		evalManager := createEvalManager(memoryStorage, log)
		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, nil, evalManager, nil, nil)

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

		evalManager := createEvalManager(memoryStorage, log)
		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient, inputBuilder, evalManager, nil, rvc)

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

		err := memoryStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixOrchestrationID, State: orchestration.Canceled})
		require.NoError(t, err)

		upgradeOperation := fixUpgradeKymaOperation()
		err = memoryStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		provisioningOperation := fixProvisioningOperation()
		err = memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
		require.NoError(t, err)

		evalManager := createEvalManager(memoryStorage, log)
		step := NewInitialisationStep(memoryStorage.Operations(), memoryStorage.Instances(), nil, nil, evalManager, nil, nil)

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
}

func fixUpgradeKymaOperation() internal.UpgradeKymaOperation {
	n := time.Now()
	windowEnd := n.Add(time.Minute)
	return internal.UpgradeKymaOperation{
		Operation: internal.Operation{
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
