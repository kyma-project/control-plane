package storage_test

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/postsql"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/pivotal-cf/brokerapi/v7/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixInstanceId = "inst-id"
)

func TestPostgres(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	t.Run("Init tests", func(t *testing.T) {
		t.Run("Should initialize database when schema not applied", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			// when
			connection, err := postsql.InitializeDatabase(cfg.ConnectionURL(), 1, logrus.New())

			require.NoError(t, err)
			require.NotNil(t, connection)

			defer storage.CloseDatabase(t, connection)

			// then
			assert.NoError(t, err)
		})

		t.Run("Should return error when failed to connect to the database", func(t *testing.T) {
			containerCleanupFunc, _, err := storage.InitTestDBContainer(t, ctx, "test_DB_3")
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
	t.Run("Instances", func(t *testing.T) {
		t.Run("Should create and update instance", func(t *testing.T) {
			// given
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			// when
			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())

			require.NoError(t, err)
			require.NotNil(t, brokerStorage)

			// given

			testData := "test"
			instanceData := instanceData{val: testData}

			fixInstance := fixInstance(instanceData)
			err = brokerStorage.Instances().Insert(*fixInstance)
			require.NoError(t, err)

			fixInstance.DashboardURL = "diff"
			_, err = brokerStorage.Instances().Update(*fixInstance)
			require.NoError(t, err)

			err = brokerStorage.Operations().InsertProvisioningOperation(internal.ProvisioningOperation{
				Operation: internal.Operation{
					InstanceDetails: internal.InstanceDetails{
						Lms: internal.LMS{
							TenantID: "lms-tenant-id",
						},
						SubAccountID: instanceData.subAccountID,
					},
					ID:                     "op-id",
					Version:                0,
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now().Add(time.Second),
					InstanceID:             fixInstance.InstanceID,
					ProvisionerOperationID: "provisioner-op-id",
					State:                  domain.Succeeded,
				},
			})
			err = brokerStorage.Operations().InsertProvisioningOperation(internal.ProvisioningOperation{
				Operation: internal.Operation{
					InstanceDetails: internal.InstanceDetails{
						Lms: internal.LMS{
							TenantID: "lms-tenant-id",
						},
						SubAccountID: instanceData.subAccountID,
					},
					ID:                     "latest-op-id",
					Version:                0,
					CreatedAt:              time.Now().Add(time.Minute),
					UpdatedAt:              time.Now().Add(2 * time.Minute),
					InstanceID:             fixInstance.InstanceID,
					ProvisionerOperationID: "provisioner-op-id",
					State:                  domain.Succeeded,
				},
			})
			require.NoError(t, err)
			err = brokerStorage.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
				Operation: internal.Operation{
					ID:    "operation-id-3",
					State: orchestration.Pending,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Hour),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute),
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "pending-operation",
					Version:                1,
					OrchestrationID:        "orch-id",
				},
				RuntimeOperation: fixRuntimeOperation("operation-id-3"),
			})
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
			assert.Equal(t, fixInstance.Parameters, inst.Parameters)
			assert.Equal(t, "lms-tenant-id", inst.InstanceDetails.Lms.TenantID)
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			psqlStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			psqlStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			psqlStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			psqlStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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

			assertInstanceByIgnoreTime(t, fixInstances[0], out[0])
			assertInstanceByIgnoreTime(t, fixInstances[1], out[1])

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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			psqlStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
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
					InstanceID:             fixInstanceId,
					OrchestrationID:        orchestrationID,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					ProvisioningParameters: fixProvisioningParameters(),
					InstanceDetails:        fixInstanceDetails(),
				},
			}
			latestOperation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:    "latest-id",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Minute),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Minute),
					InstanceID:             fixInstanceId,
					OrchestrationID:        orchestrationID,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					ProvisioningParameters: fixProvisioningParameters(),
					InstanceDetails:        fixInstanceDetails(),
				},
			}
			latestPendingOperation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:    "latest-id-pending",
					State: orchestration.Pending,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Minute),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(3 * time.Minute),
					InstanceID:             fixInstanceId,
					OrchestrationID:        orchestrationID,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					ProvisioningParameters: fixProvisioningParameters(),
					InstanceDetails:        fixInstanceDetails(),
				},
			}

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
			require.NoError(t, err)

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

			ops, err := svc.GetNotFinishedOperationsByType(dbmodel.OperationTypeProvision)
			require.NoError(t, err)
			assert.Len(t, ops, 3)
			assertOperation(t, givenOperation.Operation, ops[0])

			gotOperation, err := svc.GetProvisioningOperationByID("operation-id")
			require.NoError(t, err)

			op, err := svc.GetOperationByID("operation-id")
			require.NoError(t, err)
			assert.Equal(t, givenOperation.Operation.ID, op.ID)

			lastOp, err := svc.GetLastOperation(fixInstanceId)
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
			opList, err := svc.ListProvisioningOperationsByInstanceID(fixInstanceId)
			// then
			require.NoError(t, err)
			assert.Equal(t, 3, len(opList))
		})
		t.Run("Deprovisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.DeprovisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-id",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(time.Second),
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
				},
			}

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
			require.NoError(t, err)

			svc := brokerStorage.Operations()

			// when
			err = svc.InsertDeprovisioningOperation(givenOperation)
			require.NoError(t, err)

			ops, err := svc.GetNotFinishedOperationsByType(dbmodel.OperationTypeDeprovision)
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
					InstanceID: fixInstanceId,
					CreatedAt:  time.Now().Add(1 * time.Hour),
					UpdatedAt:  time.Now().Add(1 * time.Hour),
				},
			})
			require.NoError(t, err)
			// when
			opList, err := svc.ListDeprovisioningOperationsByInstanceID(fixInstanceId)
			// then
			require.NoError(t, err)
			assert.Equal(t, 2, len(opList))

		})
		t.Run("Upgrade", func(t *testing.T) {
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
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
					InstanceID:             fixInstanceId,
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
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					Version:                1,
					OrchestrationID:        orchestrationID,
					ProvisioningParameters: internal.ProvisioningParameters{},
				},
				RuntimeOperation: fixRuntimeOperation("operation-id-3"),
			}

			givenOperation3 := internal.UpgradeKymaOperation{
				Operation: internal.Operation{
					ID:    "operation-id-3",
					State: orchestration.Pending,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Hour),
					UpdatedAt:              time.Now().Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute),
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "pending-operation",
					Version:                1,
					OrchestrationID:        orchestrationID,
				},
				RuntimeOperation: fixRuntimeOperation("operation-id-3"),
			}

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
			require.NoError(t, err)

			svc := brokerStorage.Operations()

			// when
			err = svc.InsertUpgradeKymaOperation(givenOperation1)
			require.NoError(t, err)
			err = svc.InsertUpgradeKymaOperation(givenOperation2)
			require.NoError(t, err)
			err = svc.InsertUpgradeKymaOperation(givenOperation3)
			require.NoError(t, err)

			op, err := svc.GetUpgradeKymaOperationByInstanceID(fixInstanceId)
			require.NoError(t, err)

			lastOp, err := svc.GetLastOperation(fixInstanceId)
			require.NoError(t, err)
			assert.Equal(t, givenOperation2.Operation.ID, lastOp.ID)

			assertUpgradeKymaOperation(t, givenOperation3, *op)

			ops, count, totalCount, err := svc.ListUpgradeKymaOperationsByOrchestrationID(orchestrationID, dbmodel.OperationFilter{PageSize: 10, Page: 1})
			require.NoError(t, err)
			assert.Len(t, ops, 3)
			assert.Equal(t, count, 3)
			assert.Equal(t, totalCount, 3)
		})
	})
	t.Run("Operations conflicts", func(t *testing.T) {
		t.Run("Provisioning", func(t *testing.T) {
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-001",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now().Add(time.Second),
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
					ProvisioningParameters: internal.ProvisioningParameters{},
					InstanceDetails: internal.InstanceDetails{
						Lms: internal.LMS{TenantID: "tenant-id"},
					},
				},
			}

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
			containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
			require.NoError(t, err)
			defer containerCleanupFunc()

			givenOperation := internal.DeprovisioningOperation{
				Operation: internal.Operation{
					ID:    "operation-001",
					State: domain.InProgress,
					// used Round and set timezone to be able to compare timestamps
					CreatedAt:              time.Now(),
					UpdatedAt:              time.Now().Add(time.Second),
					InstanceID:             fixInstanceId,
					ProvisionerOperationID: "target-op-id",
					Description:            "description",
				},
			}

			err = storage.InitTestDBTables(t, cfg.ConnectionURL())
			require.NoError(t, err)

			cipher := storage.NewEncrypter(cfg.SecretKey)
			brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
	t.Run("Conflict Instances", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)

		svc := brokerStorage.Instances()

		inst := internal.Instance{
			InstanceID:      "abcd-01234",
			RuntimeID:       "r-id-001",
			GlobalAccountID: "ga-001",
			SubAccountID:    "sa-001",
			ServiceID:       "service-id-001",
			ServiceName:     "awesome-service",
			ServicePlanID:   "plan-id",
			ServicePlanName: "awesome-plan",
			DashboardURL:    "",
			Parameters:      internal.ProvisioningParameters{},
			ProviderRegion:  "",
			CreatedAt:       time.Now(),
			Version:         0,
		}

		err = svc.Insert(inst)
		require.NoError(t, err)

		// try an update
		inst.DashboardURL = "http://kyma.org"
		newInst, err := svc.Update(inst)
		require.NoError(t, err)

		// try another update with old version - expect conflict
		inst.DashboardURL = "---"
		_, err = svc.Update(inst)
		require.Error(t, err)
		assert.True(t, dberr.IsConflict(err))

		// try second update with correct version
		newInst.DashboardURL = "http://new.kyma.com"
		_, err = svc.Update(*newInst)
		require.NoError(t, err)
	})
	t.Run("Orchestrations", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
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

		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
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

		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()

		lmsTenant := internal.LMSTenant{
			ID:     "tenant-001",
			Region: "na",
			Name:   "some-company",
		}
		err = storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
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
	assert.Equal(t, expected.ProvisioningParameters, got.ProvisioningParameters)
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
	expected.ProvisioningParameters = got.ProvisioningParameters
	assert.Equal(t, expected, got)
}

func assertDeprovisioningOperation(t *testing.T, expected, got internal.DeprovisioningOperation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

	expected.CreatedAt = got.CreatedAt
	expected.UpdatedAt = got.UpdatedAt
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
	assert.Equal(t, expected, got)
}

func assertOperation(t *testing.T, expected, got internal.Operation) {
	// do not check zones and monothonic clock, see: https://golang.org/pkg/time/#Time
	assert.True(t, expected.CreatedAt.Equal(got.CreatedAt), fmt.Sprintf("Expected %s got %s", expected.CreatedAt, got.CreatedAt))
	assert.Equal(t, expected.InstanceDetails, got.InstanceDetails)

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

	instance := internal.FixInstance(testData.val)
	instance.GlobalAccountID = gaid
	instance.SubAccountID = suid
	instance.ServiceID = testData.val
	instance.ServiceName = testData.val
	instance.ServicePlanID = testData.val
	instance.ServicePlanName = testData.val
	instance.DashboardURL = fmt.Sprintf("https://console.%s.kyma.local", testData.val)
	instance.ProviderRegion = testData.val
	instance.Parameters.ErsContext.SubAccountID = suid
	instance.Parameters.ErsContext.GlobalAccountID = gaid

	return &instance
}

func fixProvisionOperation(testData string) internal.ProvisioningOperation {
	operationId := fmt.Sprintf("%s-%d", testData, rand.Int())
	return internal.FixProvisioningOperation(operationId, testData)

}
func fixDeprovisionOperation(testData string) internal.DeprovisioningOperation {
	operationId := fmt.Sprintf("%s-%d", testData, rand.Int())
	return internal.FixDeprovisioningOperation(operationId, testData)
}

func fixRuntimeOperation(operationId string) orchestration.RuntimeOperation {
	runtime := orchestration.Runtime{
		ShootName:              "shoot-stage",
		MaintenanceWindowBegin: time.Now().Truncate(time.Millisecond).Add(time.Hour),
		MaintenanceWindowEnd:   time.Now().Truncate(time.Millisecond).Add(time.Minute).Add(time.Hour),
		RuntimeID:              "runtime-id",
		GlobalAccountID:        "global-account-if",
		SubAccountID:           "subaccount-id",
	}

	runtimeOperation := internal.FixRuntimeOperation(operationId)
	runtimeOperation.Runtime = runtime

	return runtimeOperation
}

func fixProvisioningParameters() internal.ProvisioningParameters {
	active := true
	return internal.ProvisioningParameters{
		PlanID:    broker.TrialPlanID,
		ServiceID: broker.KymaServiceID,
		ErsContext: internal.ERSContext{
			TenantID:        "test",
			SubAccountID:    "test",
			GlobalAccountID: "test",
			ServiceManager: &internal.ServiceManagerEntryDTO{
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: "username",
						Password: "password",
					}},
			},
			Active: &active,
		},
		Parameters: internal.ProvisioningParametersDTO{
			Name:        "test",
			KymaVersion: "0.0.0",
		},
		PlatformRegion: "region",
	}
}

func fixInstanceDetails() internal.InstanceDetails {
	return internal.InstanceDetails{
		Lms: internal.LMS{
			TenantID: "tenant-id",
		},
		SubAccountID: "test",
		RuntimeID:    "test",
		ShootName:    "test",
		ShootDomain:  "test",
	}
}
