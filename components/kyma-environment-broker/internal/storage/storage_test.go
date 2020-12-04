// +build database_integration

package storage

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaInitializer(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("Init tests", func(t *testing.T) {
		t.Run("Should initialize database when schema not applied", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			// when
			connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), 1, logrus.New())

			require.NoError(t, err)
			require.NotNil(t, connection)

			defer CloseDatabase(t, connection)

			// then
			assert.NoError(t, err)
		})

		t.Run("Should return error when failed to connect to the database", func(t *testing.T) {
			containerCleanupFunc, _, err := InitTestDBContainer(t, ctx, "test_DB_3")
			require.NoError(t, err)
			defer containerCleanupFunc()

			// given
			connString := "bad connection string"

			// when
			connection, err := postsql.InitializeDatabase(connString, 1, logrus.New())

			// then
			assert.Error(t, err)
			assert.Nil(t, connection)
		})
	})

	// Instances
	t.Run("Instances", func(t *testing.T) {
		t.Run("Should create and update instance", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			// when
			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)

			testData := "test"
			instanceData := instanceData{val: testData}

			fixInstance := fixInstance(instanceData)
			err = brokerStorage.Instances().Insert(*fixInstance)
			require.NoError(t, err)

			fixInstance.DashboardURL = "diff"
			err = brokerStorage.Instances().Update(*fixInstance)
			require.NoError(t, err)

			// then
			inst, err := brokerStorage.Instances().GetByID(testData)
			assert.NoError(t, err)
			require.NotNil(t, inst)

			assert.Equal(t, fixInstance.InstanceID, inst.InstanceID)
			assert.Equal(t, fixInstance.RuntimeID, inst.RuntimeID)
			assert.Equal(t, fixInstance.GlobalAccountID, inst.GlobalAccountID)
			assert.Equal(t, fixInstance.ServiceID, inst.ServiceID)
			assert.Equal(t, fixInstance.ServicePlanID, inst.ServicePlanID)
			assert.Equal(t, fixInstance.DashboardURL, inst.DashboardURL)
			assert.Equal(t, fixInstance.ProvisioningParameters, inst.ProvisioningParameters)
			assert.NotEmpty(t, inst.CreatedAt)
			assert.NotEmpty(t, inst.UpdatedAt)
			assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", inst.DeletedAt.String())

			// when
			err = brokerStorage.Instances().Delete(fixInstance.InstanceID)

			// then
			assert.NoError(t, err)
			_, err = brokerStorage.Instances().GetByID(fixInstance.InstanceID)
			assert.True(t, dberr.IsNotFound(err))

			// when
			err = brokerStorage.Instances().Delete(fixInstance.InstanceID)
			assert.NoError(t, err, "deletion non existing instance must not cause any error")
		})

		t.Run("Should fetch instance statistics", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			psqlStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)
			require.NotNil(t, psqlStorage)

			// populate database with samples
			fixInstances := []internal.Instance{
				*fixInstance(instanceData{val: "A1", globalAccountID: "A"}),
				*fixInstance(instanceData{val: "A2", globalAccountID: "A"}),
				*fixInstance(instanceData{val: "C1", globalAccountID: "C"}),
			}
			for _, i := range fixInstances {
				err = psqlStorage.Instances().Insert(i)
				require.NoError(t, err)
			}

			// when
			stats, err := psqlStorage.Instances().GetInstanceStats()
			require.NoError(t, err)
			numberOfInstancesA, err := psqlStorage.Instances().GetNumberOfInstancesForGlobalAccountID("A")
			require.NoError(t, err)
			numberOfInstancesC, err := psqlStorage.Instances().GetNumberOfInstancesForGlobalAccountID("C")
			require.NoError(t, err)

			t.Logf("%+v", stats)

			// then
			assert.Equal(t, internal.InstanceStats{
				TotalNumberOfInstances: 3,
				PerGlobalAccountID:     map[string]int{"A": 2, "C": 1},
			}, stats)
			assert.Equal(t, 2, numberOfInstancesA)
			assert.Equal(t, 1, numberOfInstancesC)

		})

		t.Run("Should fetch instances along with their operations", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			psqlStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)
			require.NotNil(t, psqlStorage)

			// populate database with samples
			fixInstances := []internal.Instance{*fixInstance(instanceData{val: "A1"}), *fixInstance(instanceData{val: "B1"}), *fixInstance(instanceData{val: "C1"})}
			for _, i := range fixInstances {
				err = psqlStorage.Instances().Insert(i)
				require.NoError(t, err)
			}

			fixProvisionOp := []internal.ProvisioningOperation{fixProvisionOperation("A1"), fixProvisionOperation("B1"), fixProvisionOperation("C1")}
			for _, op := range fixProvisionOp {
				err = psqlStorage.Operations().InsertProvisioningOperation(op)
				require.NoError(t, err)
			}

			fixDeprovisionOp := []internal.DeprovisioningOperation{fixDeprovisionOperation("A1"), fixDeprovisionOperation("B1"), fixDeprovisionOperation("C1")}
			for _, op := range fixDeprovisionOp {
				err = psqlStorage.Operations().InsertDeprovisioningOperation(op)
				require.NoError(t, err)
			}

			// then
			out, err := psqlStorage.Instances().FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
			require.NoError(t, err)

			require.Len(t, out, 6)

			//  checks order of instance, the oldest should be first
			sorted := sort.SliceIsSorted(out, func(i, j int) bool {
				return out[i].CreatedAt.Before(out[j].CreatedAt)
			})
			assert.True(t, sorted)

			// ignore time as this is set internally by database so will be different
			assertInstanceByIgnoreTime(t, fixInstances[0], out[0].Instance)
			assertInstanceByIgnoreTime(t, fixInstances[0], out[1].Instance)
			assertInstanceByIgnoreTime(t, fixInstances[1], out[2].Instance)
			assertInstanceByIgnoreTime(t, fixInstances[1], out[3].Instance)
			assertInstanceByIgnoreTime(t, fixInstances[2], out[4].Instance)
			assertInstanceByIgnoreTime(t, fixInstances[2], out[5].Instance)

			assertEqualOperation(t, fixProvisionOp[0], out[0])
			assertEqualOperation(t, fixDeprovisionOp[0], out[1])
			assertEqualOperation(t, fixProvisionOp[1], out[2])
			assertEqualOperation(t, fixDeprovisionOp[1], out[3])
			assertEqualOperation(t, fixProvisionOp[2], out[4])
			assertEqualOperation(t, fixDeprovisionOp[2], out[5])
		})

		t.Run("Should fetch instances based on subaccount list", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			psqlStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)
			require.NotNil(t, psqlStorage)

			// populate database with samples
			subaccountList := []string{"sa1", "sa2", "sa3"}
			fixInstances := []internal.Instance{
				*fixInstance(instanceData{val: "1", subAccountID: subaccountList[0]}),
				*fixInstance(instanceData{val: "2", subAccountID: "someSU"}),
				*fixInstance(instanceData{val: "3", subAccountID: subaccountList[1]}),
				*fixInstance(instanceData{val: "4", subAccountID: subaccountList[2]}),
			}
			for _, i := range fixInstances {
				err = psqlStorage.Instances().Insert(i)
				require.NoError(t, err)
			}

			// when
			out, err := psqlStorage.Instances().FindAllInstancesForSubAccounts(subaccountList)

			// then
			require.NoError(t, err)
			require.Len(t, out, 3)

			require.Contains(t, []string{"1", "3", "4"}, out[0].InstanceID)
			require.Contains(t, []string{"1", "3", "4"}, out[1].InstanceID)
			require.Contains(t, []string{"1", "3", "4"}, out[2].InstanceID)
		})

		t.Run("should list instances based on page and page size", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			psqlStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)
			require.NotNil(t, psqlStorage)

			// populate database with samples
			fixInstances := []internal.Instance{
				*fixInstance(instanceData{val: "1"}),
				*fixInstance(instanceData{val: "2"}),
				*fixInstance(instanceData{val: "3"}),
			}
			for _, i := range fixInstances {
				err = psqlStorage.Instances().Insert(i)
				require.NoError(t, err)
			}
			// when
			out, count, totalCount, err := psqlStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 1})

			// then
			require.NoError(t, err)
			require.Equal(t, 2, count)
			require.Equal(t, 3, totalCount)

			assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)
			assert.Equal(t, fixInstances[1].InstanceID, out[1].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 2})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 3, totalCount)

			assert.Equal(t, fixInstances[2].InstanceID, out[0].InstanceID)
		})

		t.Run("should list instances based on filters", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			psqlStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)
			require.NotNil(t, psqlStorage)

			// populate database with samples
			fixInstances := []internal.Instance{
				*fixInstance(instanceData{val: "inst1"}),
				*fixInstance(instanceData{val: "inst2"}),
				*fixInstance(instanceData{val: "inst3"}),
			}
			for _, i := range fixInstances {
				err = psqlStorage.Instances().Insert(i)
				require.NoError(t, err)
			}
			// when
			out, count, totalCount, err := psqlStorage.Instances().List(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{RuntimeIDs: []string{fixInstances[1].RuntimeID}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{Plans: []string{fixInstances[1].ServicePlanName}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{Domains: []string{"inst2"}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

			// when
			out, count, totalCount, err = psqlStorage.Instances().List(dbmodel.InstanceFilter{Regions: []string{"inst2"}})

			// then
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Equal(t, 1, totalCount)

			assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)
		})
	})

	t.Run("Operations", func(t *testing.T) {
		t.Run("Provisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			orchestrationID := "orch-id"
			givenOperation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-id",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Second),
					InstanceID:             "inst-id",
					OrchestrationID:        orchestrationID,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
				},
				Lms:                    internal.LMS{TenantID: "tenant-id"},
				ProvisioningParameters: `{"k":"v"}`,
			}

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)

			err = brokerStorage.Orchestrations().Insert(internal.Orchestration{OrchestrationID: orchestrationID})
			require.NoError(t, err)

			svc := brokerStorage.Operations()

			// when
			err = svc.InsertProvisioningOperation(givenOperation)
			require.NoError(t, err)

			ops, err := svc.GetOperationsInProgressByType(dbmodel.OperationTypeProvision)
			require.NoError(t, err)
			assert.Len(t, ops, 1)
			assertOperation(t, givenOperation.Operation, ops[0])

			gotOperation, err := svc.GetProvisioningOperationByID("operation-id")
			require.NoError(t, err)

			op, err := svc.GetOperationByID("operation-id")
			require.NoError(t, err)
			assert.Equal(t, givenOperation.Operation.ID, op.ID)

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
			stats, err := svc.GetOperationStats()
			require.NoError(t, err)

			assert.Equal(t, 1, stats.Provisioning[domain.InProgress])

			opStats, err := svc.GetOperationStatsForOrchestration(orchestrationID)
			require.NoError(t, err)

			assert.Equal(t, 1, opStats[domain.InProgress])

		})

		t.Run("Deprovisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.DeprovisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-id",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Second),
					InstanceID:             "inst-id",
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
				},
			}

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)

			svc := brokerStorage.Operations()

			// when
			err = svc.InsertDeprovisioningOperation(givenOperation)
			require.NoError(t, err)

			ops, err := svc.GetOperationsInProgressByType(dbmodel.OperationTypeDeprovision)
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

		})
		t.Run("Upgrade", func(t *testing.T) {
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			orchestrationID := "orchestration-id"
			givenOperation1 := internal.UpgradeKymaOperation{
				Operation: internal.Operation{
					ID:    "operation-id-1",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Second),
					InstanceID:             "inst-id",
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					OrchestrationID:        orchestrationID,
				},
				RuntimeOperation: orchestration.RuntimeOperation{},
			}

			givenOperation2 := internal.UpgradeKymaOperation{
				Operation: internal.Operation{
					ID:    "operation-id-2",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Minute),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Second).Add(time.Minute),
					InstanceID:             "inst-id",
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					OrchestrationID:        orchestrationID,
				},
				RuntimeOperation: orchestration.RuntimeOperation{
					ID: "operation-id-2",
					Runtime: orchestration.Runtime{
						ShootName:              "shoot-stage",
						MaintenanceWindowBegin: time.Now().Truncate(time.Millisecond).Add(time.Hour),
						MaintenanceWindowEnd:   time.Now().Truncate(time.Millisecond).Add(time.Minute).Add(time.Hour),
						RuntimeID:              "runtime-id",
						GlobalAccountID:        "global-account-if",
						SubAccountID:           "subaccount-id",
					},
					DryRun: false,
				},
				ProvisioningParameters: "{}",
			}

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)

			svc := brokerStorage.Operations()

			// when
			err = svc.InsertUpgradeKymaOperation(givenOperation1)
			require.NoError(t, err)
			err = svc.InsertUpgradeKymaOperation(givenOperation2)
			require.NoError(t, err)

			op, err := svc.GetUpgradeKymaOperationByInstanceID("inst-id")
			require.NoError(t, err)

			assertUpgradeKymaOperation(t, givenOperation2, *op)

			ops, count, totalCount, err := svc.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})
			require.NoError(t, err)
			assert.Len(t, ops, 2)
			assert.Equal(t, count, 2)
			assert.Equal(t, totalCount, 2)
		})
	})

	t.Run("Operations conflicts", func(t *testing.T) {
		t.Run("Provisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-001",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now().Add(time.Second),
					InstanceID:             "inst-id",
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
				},
				Lms:                    internal.LMS{TenantID: "tenant-id"},
				ProvisioningParameters: `{"key":"value"}`,
			}

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			svc := brokerStorage.Provisioning()

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)
			err = svc.InsertProvisioningOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetProvisioningOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetProvisioningOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = "new modified description 1"
			gotOperation2.Description = "new modified description 2"
			_, err = svc.UpdateProvisioningOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateProvisioningOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertProvisioningOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})
		t.Run("Deprovisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.DeprovisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-001",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now().Add(time.Second),
					InstanceID:             "inst-id",
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
				},
			}

			err = InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
			require.NoError(t, err)

			svc := brokerStorage.Deprovisioning()

			err = svc.InsertDeprovisioningOperation(givenOperation)
			require.NoError(t, err)

			// when
			gotOperation1, err := svc.GetDeprovisioningOperationByID("operation-001")
			require.NoError(t, err)

			gotOperation2, err := svc.GetDeprovisioningOperationByID("operation-001")
			require.NoError(t, err)

			// when
			gotOperation1.Description = "new modified description 1"
			gotOperation2.Description = "new modified description 2"
			_, err = svc.UpdateDeprovisioningOperation(*gotOperation1)
			require.NoError(t, err)

			_, err = svc.UpdateDeprovisioningOperation(*gotOperation2)

			// then
			assertError(t, dberr.CodeConflict, err)

			// when
			err = svc.InsertDeprovisioningOperation(*gotOperation1)

			// then
			assertError(t, dberr.CodeAlreadyExists, err)
		})
	})

	t.Run("Orchestrations", func(t *testing.T) {
		containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		now := time.Now()

		const fixID = "test"
		givenOrchestration := internal.Orchestration{
			OrchestrationID: fixID,
			State:           "test",
			Description:     "test",
			CreatedAt:       now,
			UpdatedAt:       now,
			Parameters: orchestration.Parameters{
				DryRun: true,
			},
		}

		err = InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
		require.NoError(t, err)

		svc := brokerStorage.Orchestrations()

		err = svc.Insert(givenOrchestration)
		require.NoError(t, err)

		// when
		gotOrchestration, err := svc.GetByID(fixID)
		require.NoError(t, err)
		assert.Equal(t, givenOrchestration.Parameters, gotOrchestration.Parameters)

		gotOrchestration.Description = "new modified description 1"
		err = svc.Update(givenOrchestration)
		require.NoError(t, err)

		err = svc.Insert(givenOrchestration)
		assertError(t, dberr.CodeAlreadyExists, err)

		l, count, totalCount, err := svc.List(dbmodel.OrchestrationFilter{PageSize: 10, Page: 1})
		require.NoError(t, err)
		assert.Len(t, l, 1)
		assert.Equal(t, 1, count)
		assert.Equal(t, 1, totalCount)

		l, err = svc.ListByState("test")
		require.NoError(t, err)
		assert.Len(t, l, 1)
	})

	t.Run("RuntimeStates", func(t *testing.T) {
		containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		fixID := "test"
		givenRuntimeState := internal.RuntimeState{
			ID:          fixID,
			CreatedAt:   time.Now(),
			RuntimeID:   fixID,
			OperationID: fixID,
			KymaConfig: gqlschema.KymaConfigInput{
				Version: fixID,
			},
			ClusterConfig: gqlschema.GardenerConfigInput{
				KubernetesVersion: fixID,
			},
		}

		err = InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
		require.NoError(t, err)

		svc := brokerStorage.RuntimeStates()

		err = svc.Insert(givenRuntimeState)
		require.NoError(t, err)

		runtimeStates, err := svc.ListByRuntimeID(fixID)
		require.NoError(t, err)
		assert.Len(t, runtimeStates, 1)
		assert.Equal(t, fixID, runtimeStates[0].KymaConfig.Version)
		assert.Equal(t, fixID, runtimeStates[0].ClusterConfig.KubernetesVersion)

		state, err := svc.GetByOperationID(fixID)
		require.NoError(t, err)
		assert.Equal(t, fixID, state.KymaConfig.Version)
		assert.Equal(t, fixID, state.ClusterConfig.KubernetesVersion)
	})

	t.Run("LMS Tenants", func(t *testing.T) {
		containerCleanupFunc, cfg, err := InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		lmsTenant := internal.LMSTenant{
			ID:     "tenant-001",
			Region: "na",
			Name:   "some-company",
		}
		err = InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		brokerStorage, _, err := NewFromConfig(cfg, logrus.StandardLogger())
		svc := brokerStorage.LMSTenants()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		// when
		err = svc.InsertTenant(lmsTenant)
		require.NoError(t, err)
		gotTenant, found, err := svc.FindTenantByName("some-company", "na")
		_, differentRegionExists, drErr := svc.FindTenantByName("some-company", "us")
		_, differentNameExists, dnErr := svc.FindTenantByName("some-company1", "na")

		// then
		assert.Equal(t, lmsTenant.Name, gotTenant.Name)
		assert.True(t, found)
		assert.NoError(t, err)
		assert.False(t, differentRegionExists)
		assert.NoError(t, drErr)
		assert.False(t, differentNameExists)
		assert.NoError(t, dnErr)
	})
}

func assertProvisioningOperation(t *testing.T, expected, got internal.ProvisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.JSONEq(t, expected.ProvisioningParameters, got.ProvisioningParameters)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.ProvisioningParameters = got.ProvisioningParameters
	assert.Equal(t, expected, got)
}

func assertDeprovisioningOperation(t *testing.T, expected, got internal.DeprovisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	assert.Equal(t, expected, got)
}

func assertUpgradeKymaOperation(t *testing.T, expected, got internal.UpgradeKymaOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.True(t, expected.MaintenanceWindowBegin.Equal(got.MaintenanceWindowBegin))
	assert.True(t, expected.MaintenanceWindowEnd.Equal(got.MaintenanceWindowEnd))

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.MaintenanceWindowBegin = got.MaintenanceWindowBegin
	expected.MaintenanceWindowEnd = got.MaintenanceWindowEnd
	assert.Equal(t, expected, got)
}

func assertOperation(t *testing.T, expected, got internal.Operation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	assert.Equal(t, expected, got)
}

func assertError(t *testing.T, expectedCode int, err error) {
	require.Error(t, err)

	dbe, ok := err.(dberr.Error)
	if !ok {
		assert.Fail(t, "expected DB Error Conflict")
	}
	assert.Equal(t, expectedCode, dbe.Code())
}

func assertInstanceByIgnoreTime(t *testing.T, want, got internal.Instance) {
	t.Helper()
	want.CreatedAt, got.CreatedAt = time.Time{}, time.Time{}
	want.UpdatedAt, got.UpdatedAt = time.Time{}, time.Time{}
	want.DeletedAt, got.DeletedAt = time.Time{}, time.Time{}

	assert.EqualValues(t, want, got)
}

func assertEqualOperation(t *testing.T, want interface{}, got internal.InstanceWithOperation) {
	t.Helper()
	switch want := want.(type) {
	case internal.ProvisioningOperation:
		assert.EqualValues(t, dbmodel.OperationTypeProvision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	case internal.DeprovisioningOperation:
		assert.EqualValues(t, dbmodel.OperationTypeDeprovision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	}
}

type instanceData struct {
	val             string
	globalAccountID string
	subAccountID    string
}

func fixInstance(testData instanceData) *internal.Instance {
	var (
		gaid string
		suid string
	)

	if testData.globalAccountID != "" {
		gaid = testData.globalAccountID
	} else {
		gaid = testData.val
	}

	if testData.subAccountID != "" {
		suid = testData.subAccountID
	} else {
		suid = testData.val
	}

	return &internal.Instance{
		InstanceID:      testData.val,
		RuntimeID:       testData.val,
		GlobalAccountID: gaid,
		SubAccountID:    suid,
		ServiceID:       testData.val,
		ServiceName:     testData.val,
		ServicePlanID:   testData.val,
		ServicePlanName: testData.val,
		DashboardURL:    fmt.Sprintf("https://console.%s.kyma.local", testData.val),
		ProviderRegion:  testData.val,
	}
}

func fixProvisionOperation(testData string) internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: fixSucceededOperation(testData),
	}
}
func fixDeprovisionOperation(testData string) internal.DeprovisioningOperation {
	return internal.DeprovisioningOperation{
		Operation: fixSucceededOperation(testData),
	}
}

func fixSucceededOperation(testData string) internal.Operation {
	return internal.Operation{
		ID:                     fmt.Sprintf("%s-%d", testData, rand.Int()),
		CreatedAt:              fixTime().Add(24 * time.Hour),
		UpdatedAt:              fixTime().Add(48 * time.Hour),
		InstanceID:             testData,
		ProvisionerOperationID: testData,
		State:                  domain.Succeeded,
		Description:            testData,
	}
}

func fixTime() time.Time {
	return time.Date(2020, 04, 21, 0, 0, 23, 42, time.UTC)
}
