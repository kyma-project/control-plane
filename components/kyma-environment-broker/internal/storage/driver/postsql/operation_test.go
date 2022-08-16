package postsql_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperation(t *testing.T) {

	ctx := context.Background()

	t.Run("Operations - provisioning and deprovisioning", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		orchestrationID := "orch-id"

		givenOperation := fixture.FixOperation("operation-id", "inst-id", internal.OperationTypeProvision)
		givenOperation.InputCreator = nil
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = givenOperation.CreatedAt.Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.OrchestrationID = orchestrationID
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID
		givenOperation.FinishedStagesOrdered = "prepare,check_provisioning"
		latestOperation := fixture.FixOperation("latest-id", "inst-id", internal.OperationTypeDeprovision)
		latestOperation.InputCreator = nil
		latestOperation.State = domain.InProgress
		latestOperation.CreatedAt = latestOperation.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		latestOperation.UpdatedAt = latestOperation.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestOperation.Version = 1
		latestOperation.OrchestrationID = orchestrationID
		latestOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestPendingOperation := fixture.FixOperation("latest-id-pending", "inst-id", internal.OperationTypeProvision)
		latestPendingOperation.InputCreator = nil
		latestPendingOperation.State = orchestration.Pending
		latestPendingOperation.CreatedAt = latestPendingOperation.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestPendingOperation.UpdatedAt = latestPendingOperation.UpdatedAt.Truncate(time.Millisecond).Add(3 * time.Minute)
		latestPendingOperation.Version = 1
		latestPendingOperation.OrchestrationID = orchestrationID
		latestPendingOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		err = brokerStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: orchestrationID})
		require.NoError(t, err)

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertOperation(givenOperation)
		require.NoError(t, err)
		err = svc.InsertOperation(latestOperation)
		require.NoError(t, err)
		err = svc.InsertOperation(latestPendingOperation)
		require.NoError(t, err)

		provisionOps, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeProvision)
		require.NoError(t, err)
		assert.Len(t, provisionOps, 2)
		assertOperation(t, givenOperation, provisionOps[0])

		deprovisionOps, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeDeprovision)
		require.NoError(t, err)
		assert.Len(t, deprovisionOps, 1)
		assertOperation(t, latestOperation, deprovisionOps[0])

		gotOperation, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.ID, gotOperation.ID)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, latestOperation.ID, lastOp.ID)

		latestOp, err := svc.GetOperationByInstanceID("inst-id")
		require.NoError(t, err)
		assert.Equal(t, latestPendingOperation.ID, latestOp.ID)

		// when
		opList, err := svc.ListOperationsByInstanceID("inst-id")

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, len(opList))

		// when
		_, _, totalCount, err := svc.ListOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})

		// then
		require.NoError(t, err)
		assert.Equal(t, 3, totalCount)
		assertOperation(t, givenOperation, *gotOperation)

		assertUpdateDescription(t, gotOperation, svc)

		assertUpdateState(t, svc, orchestrationID, latestOp)

		assertEmptyResultForNonExistingIds(t, svc)
	})

	t.Run("Provisioning", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		orchestrationID := "orch-id"

		givenOperation := fixture.FixProvisioningOperation("operation-id", "inst-id")
		givenOperation.InputCreator = nil
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = givenOperation.CreatedAt.Truncate(time.Millisecond)
		givenOperation.UpdatedAt = givenOperation.UpdatedAt.Truncate(time.Millisecond)
		givenOperation.Version = 1
		givenOperation.OrchestrationID = orchestrationID
		givenOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestOperation := fixture.FixProvisioningOperation("latest-id", "inst-id")
		latestOperation.InputCreator = nil
		latestOperation.State = domain.InProgress
		latestOperation.CreatedAt = latestOperation.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		latestOperation.UpdatedAt = latestOperation.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestOperation.Version = 1
		latestOperation.OrchestrationID = orchestrationID
		latestOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		latestPendingOperation := fixture.FixProvisioningOperation("latest-id-pending", "inst-id")
		latestPendingOperation.InputCreator = nil
		latestPendingOperation.State = orchestration.Pending
		latestPendingOperation.CreatedAt = latestPendingOperation.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Minute)
		latestPendingOperation.UpdatedAt = latestPendingOperation.UpdatedAt.Truncate(time.Millisecond).Add(3 * time.Minute)
		latestPendingOperation.Version = 1
		latestPendingOperation.OrchestrationID = orchestrationID
		latestPendingOperation.ProvisioningParameters.PlanID = broker.TrialPlanID

		err = brokerStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: orchestrationID})
		require.NoError(t, err)

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertProvisioningOperation(givenOperation)
		require.NoError(t, err)
		err = svc.InsertProvisioningOperation(latestOperation)
		require.NoError(t, err)
		err = svc.InsertProvisioningOperation(latestPendingOperation)
		require.NoError(t, err)

		ops, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeProvision)
		require.NoError(t, err)
		assert.Len(t, ops, 3)
		assertOperation(t, givenOperation.Operation, ops[0])

		gotOperation, err := svc.GetProvisioningOperationByID("operation-id")
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.Operation.ID, op.ID)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, latestOperation.Operation.ID, lastOp.ID)

		// then
		assertProvisioningOperation(t, givenOperation, *gotOperation)

		// when
		gotOperation.Description = "new modified description"
		_, err = svc.UpdateProvisioningOperation(*gotOperation)
		require.NoError(t, err)

		// then
		gotOperation2, err := svc.GetProvisioningOperationByID("operation-id")
		require.NoError(t, err)

		assert.Equal(t, "new modified description", gotOperation2.Description)

		// when
		stats, err := svc.GetOperationStatsByPlan()
		require.NoError(t, err)

		assert.Equal(t, 2, stats[broker.TrialPlanID].Provisioning[domain.InProgress])

		opStats, err := svc.GetOperationStatsForOrchestration(orchestrationID)
		require.NoError(t, err)

		assert.Equal(t, 2, opStats[orchestration.InProgress])

		// when
		opList, err := svc.ListProvisioningOperationsByInstanceID("inst-id")
		// then
		require.NoError(t, err)
		assert.Equal(t, 3, len(opList))
	})

	t.Run("Deprovisioning", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		givenOperation := fixture.FixDeprovisioningOperation("operation-id", "inst-id")
		givenOperation.State = domain.InProgress
		givenOperation.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(time.Second)
		givenOperation.ProvisionerOperationID = "target-op-id"
		givenOperation.Description = "description"
		givenOperation.Version = 1

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertDeprovisioningOperation(givenOperation)
		require.NoError(t, err)

		ops, err := svc.GetNotFinishedOperationsByType(internal.OperationTypeDeprovision)
		require.NoError(t, err)
		assert.Len(t, ops, 1)
		assertOperation(t, givenOperation.Operation, ops[0])

		gotOperation, err := svc.GetDeprovisioningOperationByID("operation-id")
		require.NoError(t, err)

		op, err := svc.GetOperationByID("operation-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation.Operation.ID, op.ID)

		// then
		assertDeprovisioningOperation(t, givenOperation, *gotOperation)

		// when
		gotOperation.Description = "new modified description"
		_, err = svc.UpdateDeprovisioningOperation(*gotOperation)
		require.NoError(t, err)

		// then
		gotOperation2, err := svc.GetDeprovisioningOperationByID("operation-id")
		require.NoError(t, err)

		assert.Equal(t, "new modified description", gotOperation2.Description)

		// given
		err = svc.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "other-op-id",
				InstanceID: "inst-id",
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
			},
		})
		require.NoError(t, err)
		// when
		opList, err := svc.ListDeprovisioningOperationsByInstanceID("inst-id")
		// then
		require.NoError(t, err)
		assert.Equal(t, 2, len(opList))
	})

	t.Run("Upgrade Kyma", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		orchestrationID := "orchestration-id"

		givenOperation1 := fixture.FixUpgradeKymaOperation("operation-id-1", "inst-id")
		givenOperation1.State = domain.InProgress
		givenOperation1.CreatedAt = time.Now().Truncate(time.Millisecond)
		givenOperation1.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(time.Second)
		givenOperation1.ProvisionerOperationID = "target-op-id"
		givenOperation1.Description = "description"
		givenOperation1.OrchestrationID = orchestrationID
		givenOperation1.InputCreator = nil
		givenOperation1.Version = 1

		givenOperation2 := fixture.FixUpgradeKymaOperation("operation-id-2", "inst-id")
		givenOperation2.State = domain.InProgress
		givenOperation2.CreatedAt = time.Now().Truncate(time.Millisecond).Add(time.Minute)
		givenOperation2.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(time.Second).Add(time.Minute)
		givenOperation2.ProvisionerOperationID = "target-op-id"
		givenOperation2.Description = "description"
		givenOperation2.OrchestrationID = orchestrationID
		givenOperation2.RuntimeOperation = fixRuntimeOperation("operation-id-2")
		givenOperation2.InputCreator = nil
		givenOperation2.Version = 1

		givenOperation3 := fixture.FixUpgradeKymaOperation("operation-id-3", "inst-id")
		givenOperation3.State = orchestration.Pending
		givenOperation3.CreatedAt = time.Now().Truncate(time.Millisecond).Add(2 * time.Hour)
		givenOperation3.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute)
		givenOperation3.ProvisionerOperationID = "target-op-id"
		givenOperation3.Description = "pending-operation"
		givenOperation3.OrchestrationID = orchestrationID
		givenOperation3.RuntimeOperation = fixRuntimeOperation("operation-id-3")
		givenOperation3.InputCreator = nil
		givenOperation3.Version = 1

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertUpgradeKymaOperation(givenOperation1)
		require.NoError(t, err)
		err = svc.InsertUpgradeKymaOperation(givenOperation2)
		require.NoError(t, err)
		err = svc.InsertUpgradeKymaOperation(givenOperation3)
		require.NoError(t, err)

		op, err := svc.GetUpgradeKymaOperationByInstanceID("inst-id")
		require.NoError(t, err)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastOp.ID)

		assertUpgradeKymaOperation(t, givenOperation3, *op)

		ops, count, totalCount, err := svc.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, ops, 3)
		assert.Equal(t, count, 3)
		assert.Equal(t, totalCount, 3)
	})

	t.Run("Upgrade Cluster", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		orchestrationID := "orchestration-id"

		givenOperation1 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-1", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation1.State = domain.InProgress
		givenOperation1.CreatedAt = givenOperation1.CreatedAt.Truncate(time.Millisecond)
		givenOperation1.UpdatedAt = givenOperation1.UpdatedAt.Truncate(time.Millisecond).Add(time.Second)
		givenOperation1.ProvisionerOperationID = "target-op-id"
		givenOperation1.Description = "description"
		givenOperation1.Version = 1
		givenOperation1.OrchestrationID = orchestrationID

		givenOperation2 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-2", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation2.State = domain.InProgress
		givenOperation2.CreatedAt = givenOperation2.CreatedAt.Truncate(time.Millisecond).Add(time.Minute)
		givenOperation2.UpdatedAt = givenOperation2.UpdatedAt.Truncate(time.Millisecond).Add(time.Minute).Add(time.Second)
		givenOperation2.ProvisionerOperationID = "target-op-id"
		givenOperation2.Description = "description"
		givenOperation2.Version = 1
		givenOperation2.OrchestrationID = orchestrationID
		givenOperation2.RuntimeOperation = fixRuntimeOperation("operation-id-2")

		givenOperation3 := internal.UpgradeClusterOperation{
			Operation: fixture.FixOperation("operation-id-3", "inst-id", internal.OperationTypeUpgradeCluster),
		}
		givenOperation3.State = orchestration.Pending
		givenOperation3.CreatedAt = givenOperation3.CreatedAt.Truncate(time.Millisecond).Add(2 * time.Hour)
		givenOperation3.UpdatedAt = givenOperation3.UpdatedAt.Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute)
		givenOperation3.ProvisionerOperationID = "target-op-id"
		givenOperation3.Description = "pending-operation"
		givenOperation3.Version = 1
		givenOperation3.OrchestrationID = orchestrationID
		givenOperation3.RuntimeOperation = fixRuntimeOperation("operation-id-3")

		svc := brokerStorage.Operations()

		// when
		err = svc.InsertUpgradeClusterOperation(givenOperation1)
		require.NoError(t, err)
		err = svc.InsertUpgradeClusterOperation(givenOperation2)
		require.NoError(t, err)
		err = svc.InsertUpgradeClusterOperation(givenOperation3)
		require.NoError(t, err)

		// then
		op, err := svc.GetUpgradeClusterOperationByID(givenOperation3.Operation.ID)
		require.NoError(t, err)
		assertUpgradeClusterOperation(t, givenOperation3, *op)

		lastOp, err := svc.GetLastOperation("inst-id")
		require.NoError(t, err)
		assert.Equal(t, givenOperation2.Operation.ID, lastOp.ID)

		ops, count, totalCount, err := svc.ListUpgradeClusterOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, ops, 3)
		assert.Equal(t, count, 3)
		assert.Equal(t, totalCount, 3)

		ops, err = svc.ListUpgradeClusterOperationsByInstanceID("inst-id")
		require.NoError(t, err)
		assert.Len(t, ops, 3)

		// when
		givenOperation3.Description = "diff"
		givenOperation3.ProvisionerOperationID = "modified-op-id"
		op, err = svc.UpdateUpgradeClusterOperation(givenOperation3)
		op.CreatedAt = op.CreatedAt.Truncate(time.Millisecond)
		op.MaintenanceWindowBegin = op.MaintenanceWindowBegin.Truncate(time.Millisecond)
		op.MaintenanceWindowEnd = op.MaintenanceWindowEnd.Truncate(time.Millisecond)

		// then
		got, err := svc.GetUpgradeClusterOperationByID(givenOperation3.Operation.ID)
		require.NoError(t, err)
		assertUpgradeClusterOperation(t, *op, *got)
	})
}

func assertUpdateState(t *testing.T, svc storage.Operations, orchestrationID string, latestOp *internal.Operation) {
	// when
	stats, err := svc.GetOperationStatsByPlan()
	require.NoError(t, err)

	assert.Equal(t, 1, stats[broker.TrialPlanID].Provisioning[domain.InProgress])

	opStats, err := svc.GetOperationStatsForOrchestration(orchestrationID)
	require.NoError(t, err)

	// then
	assert.Equal(t, 2, opStats[orchestration.InProgress])

	// when
	latestOp.State = domain.InProgress
	_, err = svc.UpdateOperation(*latestOp)
	opStats, err = svc.GetOperationStatsForOrchestration(orchestrationID)
	require.NoError(t, err)

	// then
	assert.Equal(t, 3, opStats[orchestration.InProgress])
}

func assertUpdateDescription(t *testing.T, gotOperation *internal.Operation, svc storage.Operations) {
	// when
	gotOperation.Description = "new modified description"
	_, err := svc.UpdateOperation(*gotOperation)
	require.NoError(t, err)

	// then
	gotOperation2, err := svc.GetOperationByID("operation-id")
	require.NoError(t, err)

	assert.Equal(t, "new modified description", gotOperation2.Description)
}

func assertEmptyResultForNonExistingIds(t *testing.T, svc storage.Operations) {
	// when
	opList, err := svc.ListOperationsByInstanceID("non-existing-inst-id")

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, len(opList))

	// when
	_, _, totalCount, err := svc.ListOperationsByOrchestrationID("non-existing-orchestration-id", dbmodel.OperationFilter{PageSize: 10, Page: 1})

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, totalCount)

	_, err = svc.GetOperationByID("non-existing-operation-id")
	require.Error(t, err, "Operation with instance_id inst-id not exist")

	_, err = svc.GetLastOperation("non-existing-inst-id")
	require.Error(t, err, "Operation with instance_id inst-id not exist")

	_, err = svc.GetOperationByInstanceID("non-existing-inst-id")
	require.Error(t, err, "operation does not exist")
}

func assertProvisioningOperation(t *testing.T, expected, got internal.ProvisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.ProvisioningParameters, got.ProvisioningParameters)
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.ProvisioningParameters = got.ProvisioningParameters
	expected.FinishedStagesOrdered = got.FinishedStagesOrdered

	assert.Equal(t, expected, got)
}

func assertDeprovisioningOperation(t *testing.T, expected, got internal.DeprovisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStagesOrdered = got.FinishedStagesOrdered

	assert.Equal(t, expected, got)
}

func assertUpgradeKymaOperation(t *testing.T, expected, got internal.UpgradeKymaOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.True(t, expected.MaintenanceWindowBegin.Equal(got.MaintenanceWindowBegin))
	assert.True(t, expected.MaintenanceWindowEnd.Equal(got.MaintenanceWindowEnd))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.MaintenanceWindowBegin = got.MaintenanceWindowBegin
	expected.MaintenanceWindowEnd = got.MaintenanceWindowEnd
	expected.FinishedStagesOrdered = got.FinishedStagesOrdered

	assert.Equal(t, expected, got)
}

func assertUpgradeClusterOperation(t *testing.T, expected, got internal.UpgradeClusterOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.True(t, expected.MaintenanceWindowBegin.Equal(got.MaintenanceWindowBegin))
	assert.True(t, expected.MaintenanceWindowEnd.Equal(got.MaintenanceWindowEnd))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.MaintenanceWindowBegin = got.MaintenanceWindowBegin
	expected.MaintenanceWindowEnd = got.MaintenanceWindowEnd
	expected.FinishedStagesOrdered = got.FinishedStagesOrdered

	assert.Equal(t, expected, got)
}

func assertOperation(t *testing.T, expected, got internal.Operation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.FinishedStagesOrdered = got.FinishedStagesOrdered

	assert.Equal(t, expected, got)
}
