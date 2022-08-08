package postsql_test

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/predicate"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstance(t *testing.T) {

	ctx := context.Background()

	t.Run("Should create and update instance", func(t *testing.T) {
		containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t.Logf, ctx, "test_DB_1")
		require.NoError(t, err)
		defer containerCleanupFunc()
		time.Hour
		tablesCleanupFunc, err := storage.InitTestDBTables(t, cfg.ConnectionURL())
		require.NoError(t, err)
		defer tablesCleanupFunc()

		cipher := storage.NewEncrypter(cfg.SecretKey)
		brokerStorage, _, err := storage.NewFromConfig(cfg, cipher, logrus.StandardLogger())
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)

		// given
		testInstanceId := "test"
		expiredID := "expired-id"
		fixInstance := fixture.FixInstance(testInstanceId)
		expiredInstance := fixture.FixInstance(expiredID)
		expiredInstance.ExpiredAt = ptr.Time(time.Now())

		err = brokerStorage.Instances().Insert(fixInstance)
		require.NoError(t, err)
		err = brokerStorage.Instances().Insert(expiredInstance)
		require.NoError(t, err)

		fixInstance.DashboardURL = "diff"
		fixInstance.Provider = "OpenStack"
		_, err = brokerStorage.Instances().Update(fixInstance)
		require.NoError(t, err)

		fixProvisioningOperation1 := fixture.FixProvisioningOperation("op-id", fixInstance.InstanceID)

		err = brokerStorage.Operations().InsertProvisioningOperation(fixProvisioningOperation1)
		require.NoError(t, err)

		fixProvisioningOperation2 := fixture.FixProvisioningOperation("latest-op-id", fixInstance.InstanceID)

		err = brokerStorage.Operations().InsertProvisioningOperation(fixProvisioningOperation2)
		require.NoError(t, err)

		upgradeOperation := fixture.FixUpgradeKymaOperation("operation-id-3", "inst-id")
		upgradeOperation.State = orchestration.Pending
		upgradeOperation.CreatedAt = time.Now().Truncate(time.Millisecond).Add(2 * time.Hour)
		upgradeOperation.UpdatedAt = time.Now().Truncate(time.Millisecond).Add(2 * time.Hour).Add(10 * time.Minute)
		upgradeOperation.ProvisionerOperationID = "target-op-id"
		upgradeOperation.Description = "pending-operation"
		upgradeOperation.Version = 1
		upgradeOperation.OrchestrationID = "orch-id"
		upgradeOperation.RuntimeOperation = fixRuntimeOperation("operation-id-3")

		err = brokerStorage.Operations().InsertUpgradeKymaOperation(upgradeOperation)
		require.NoError(t, err)

		// then
		inst, err := brokerStorage.Instances().GetByID(testInstanceId)
		assert.NoError(t, err)
		expired, err := brokerStorage.Instances().GetByID(expiredID)
		assert.NoError(t, err)
		require.NotNil(t, inst)

		assert.Equal(t, fixInstance.InstanceID, inst.InstanceID)
		assert.Equal(t, fixInstance.RuntimeID, inst.RuntimeID)
		assert.Equal(t, fixInstance.GlobalAccountID, inst.GlobalAccountID)
		assert.Equal(t, fixInstance.SubscriptionGlobalAccountID, inst.SubscriptionGlobalAccountID)
		assert.Equal(t, fixInstance.ServiceID, inst.ServiceID)
		assert.Equal(t, fixInstance.ServicePlanID, inst.ServicePlanID)
		assert.Equal(t, fixInstance.DashboardURL, inst.DashboardURL)
		assert.Equal(t, fixInstance.Parameters, inst.Parameters)
		assert.Equal(t, fixInstance.Provider, inst.Provider)
		assert.False(t, inst.IsExpired())
		assert.NotEmpty(t, inst.CreatedAt)
		assert.NotEmpty(t, inst.UpdatedAt)
		assert.Equal(t, "0001-01-01 00:00:00 +0000 UTC", inst.DeletedAt.String())
		assert.True(t, expired.IsExpired())

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

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1", globalAccountID: "A"}),
			*fixInstance(instanceData{val: "A2", globalAccountID: "A"}),
			*fixInstance(instanceData{val: "C1", globalAccountID: "C"}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// when
		stats, err := brokerStorage.Instances().GetInstanceStats()
		require.NoError(t, err)
		numberOfInstancesA, err := brokerStorage.Instances().GetNumberOfInstancesForGlobalAccountID("A")
		require.NoError(t, err)
		numberOfInstancesC, err := brokerStorage.Instances().GetNumberOfInstancesForGlobalAccountID("C")
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

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "A1"}),
			*fixInstance(instanceData{val: "B1"}),
			*fixInstance(instanceData{val: "C1"}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		fixProvisionOps := []internal.ProvisioningOperation{
			fixProvisionOperation("A1"),
			fixProvisionOperation("B1"),
			fixProvisionOperation("C1"),
		}

		for _, op := range fixProvisionOps {
			err = brokerStorage.Operations().InsertProvisioningOperation(op)
			require.NoError(t, err)
		}

		fixDeprovisionOps := []internal.DeprovisioningOperation{
			fixDeprovisionOperation("A1"),
			fixDeprovisionOperation("B1"),
			fixDeprovisionOperation("C1"),
		}

		for _, op := range fixDeprovisionOps {
			err = brokerStorage.Operations().InsertDeprovisioningOperation(op)
			require.NoError(t, err)
		}

		// then
		out, err := brokerStorage.Instances().FindAllJoinedWithOperations(predicate.SortAscByCreatedAt())
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

		assertEqualOperation(t, fixProvisionOps[0], out[0])
		assertEqualOperation(t, fixDeprovisionOps[0], out[1])
		assertEqualOperation(t, fixProvisionOps[1], out[2])
		assertEqualOperation(t, fixDeprovisionOps[1], out[3])
		assertEqualOperation(t, fixProvisionOps[2], out[4])
		assertEqualOperation(t, fixDeprovisionOps[2], out[5])
	})

	t.Run("Should fetch instances based on subaccount list", func(t *testing.T) {
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

		// populate database with samples
		subaccounts := []string{"sa1", "sa2", "sa3"}
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "1", subAccountID: subaccounts[0]}),
			*fixInstance(instanceData{val: "2", subAccountID: "someSU"}),
			*fixInstance(instanceData{val: "3", subAccountID: subaccounts[1]}),
			*fixInstance(instanceData{val: "4", subAccountID: subaccounts[2]}),
		}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// when
		out, err := brokerStorage.Instances().FindAllInstancesForSubAccounts(subaccounts)

		// then
		require.NoError(t, err)
		require.Len(t, out, 3)

		require.Contains(t, []string{"1", "3", "4"}, out[0].InstanceID)
		require.Contains(t, []string{"1", "3", "4"}, out[1].InstanceID)
		require.Contains(t, []string{"1", "3", "4"}, out[2].InstanceID)
	})

	t.Run("Should list instances based on page and page size", func(t *testing.T) {
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

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "1"}),
			*fixInstance(instanceData{val: "2"}),
			*fixInstance(instanceData{val: "3"}),
		}
		fixOperations := []internal.ProvisioningOperation{
			fixture.FixProvisioningOperation("op1", "1"),
			fixture.FixProvisioningOperation("op2", "2"),
			fixture.FixProvisioningOperation("op3", "3"),
		}
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertProvisioningOperation(i)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 1})

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 3, totalCount)

		assertInstanceByIgnoreTime(t, fixInstances[0], out[0])
		assertInstanceByIgnoreTime(t, fixInstances[1], out[1])

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{PageSize: 2, Page: 2})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 3, totalCount)

		assert.Equal(t, fixInstances[2].InstanceID, out[0].InstanceID)
	})

	t.Run("Should list instances based on filters", func(t *testing.T) {
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

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "inst1"}),
			*fixInstance(instanceData{val: "inst2"}),
			*fixInstance(instanceData{val: "inst3"}),
			*fixInstance(instanceData{val: "expiredinstance", expired: true}),
		}
		fixOperations := []internal.ProvisioningOperation{
			fixture.FixProvisioningOperation("op1", "inst1"),
			fixture.FixProvisioningOperation("op2", "inst2"),
			fixture.FixProvisioningOperation("op3", "inst3"),
			fixture.FixProvisioningOperation("op4", "expiredinstance"),
		}
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertProvisioningOperation(i)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{RuntimeIDs: []string{fixInstances[1].RuntimeID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Plans: []string{fixInstances[1].ServicePlanName}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Shoots: []string{"Shoot-inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Regions: []string{"inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(false)})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 3, totalCount)

	})

	t.Run("Should list instances based on filters", func(t *testing.T) {
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

		// populate database with samples
		fixInstances := []internal.Instance{
			*fixInstance(instanceData{val: "inst1"}),
			*fixInstance(instanceData{val: "inst2"}),
			*fixInstance(instanceData{val: "inst3"}),
			*fixInstance(instanceData{val: "expiredinstance", expired: true}),
		}
		fixOperations := []internal.ProvisioningOperation{
			fixture.FixProvisioningOperation("op1", "inst1"),
			fixture.FixProvisioningOperation("op2", "inst2"),
			fixture.FixProvisioningOperation("op3", "inst3"),
			fixture.FixProvisioningOperation("op4", "expiredinstance"),
		}
		for i, v := range fixInstances {
			v.InstanceDetails = fixture.FixInstanceDetails(v.InstanceID)
			fixInstances[i] = v
			err = brokerStorage.Instances().Insert(v)
			require.NoError(t, err)
		}
		for _, i := range fixOperations {
			err = brokerStorage.Operations().InsertProvisioningOperation(i)
			require.NoError(t, err)
		}
		// when
		out, count, totalCount, err := brokerStorage.Instances().List(dbmodel.InstanceFilter{InstanceIDs: []string{fixInstances[0].InstanceID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[0].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{GlobalAccountIDs: []string{fixInstances[1].GlobalAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{SubAccountIDs: []string{fixInstances[1].SubAccountID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{RuntimeIDs: []string{fixInstances[1].RuntimeID}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Plans: []string{fixInstances[1].ServicePlanName}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Shoots: []string{"Shoot-inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Regions: []string{"inst2"}})

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[1].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(true)})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)

		assert.Equal(t, fixInstances[3].InstanceID, out[0].InstanceID)

		// when
		out, count, totalCount, err = brokerStorage.Instances().List(dbmodel.InstanceFilter{Expired: ptr.Bool(false)})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Equal(t, 3, totalCount)

	})

	t.Run("Should list trial instances", func(t *testing.T) {
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

		// populate database with samples
		inst1 := fixInstance(instanceData{val: "inst1"})
		inst2 := fixInstance(instanceData{val: "inst2", trial: true, expired: true})
		inst3 := fixInstance(instanceData{val: "inst3", trial: true})
		inst4 := fixInstance(instanceData{val: "inst4"})
		fixInstances := []internal.Instance{*inst1, *inst2, *inst3, *inst4}

		for _, i := range fixInstances {
			err = brokerStorage.Instances().Insert(i)
			require.NoError(t, err)
		}

		// inst1 is in succeeded state
		provOp1 := fixProvisionOperation("inst1")
		provOp1.State = domain.Succeeded
		err = brokerStorage.Operations().InsertProvisioningOperation(provOp1)
		require.NoError(t, err)

		// inst2 is in error state
		provOp2 := fixProvisionOperation("inst2")
		provOp2.State = domain.Succeeded
		err = brokerStorage.Operations().InsertProvisioningOperation(provOp2)
		require.NoError(t, err)
		upgrOp2 := fixUpgradeKymaOperation("inst2")
		upgrOp2.CreatedAt = upgrOp2.CreatedAt.Add(time.Minute)
		upgrOp2.State = domain.Failed
		err = brokerStorage.Operations().InsertUpgradeKymaOperation(upgrOp2)
		require.NoError(t, err)

		// inst3 is in suspended state
		provOp3 := fixProvisionOperation("inst3")
		provOp3.State = domain.Succeeded
		err = brokerStorage.Operations().InsertProvisioningOperation(provOp3)
		require.NoError(t, err)
		upgrOp3 := fixUpgradeKymaOperation("inst3")
		upgrOp3.CreatedAt = upgrOp2.CreatedAt.Add(time.Minute)
		upgrOp3.State = domain.Failed
		err = brokerStorage.Operations().InsertUpgradeKymaOperation(upgrOp3)
		require.NoError(t, err)
		deprovOp3 := fixDeprovisionOperation("inst3")
		deprovOp3.Temporary = true
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = brokerStorage.Operations().InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)

		// inst4 is in failed state
		provOp4 := fixProvisionOperation("inst4")
		provOp4.State = domain.Failed
		err = brokerStorage.Operations().InsertProvisioningOperation(provOp4)
		require.NoError(t, err)

		// when
		nonExpiredTrialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{broker.TrialPlanID}, Expired: &[]bool{true}[0]}
		out, count, totalCount, err := brokerStorage.Instances().List(nonExpiredTrialInstancesFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Equal(t, 1, totalCount)
		require.Equal(t, inst2.InstanceID, out[0].InstanceID)

		// when
		trialInstancesFilter := dbmodel.InstanceFilter{PlanIDs: []string{broker.TrialPlanID}}
		out, count, totalCount, err = brokerStorage.Instances().List(trialInstancesFilter)

		// then
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Equal(t, 2, totalCount)
		require.Equal(t, inst2.InstanceID, out[0].InstanceID)
		require.Equal(t, inst3.InstanceID, out[1].InstanceID)
	})
}

func assertInstanceByIgnoreTime(t *testing.T, want, got internal.Instance) {
	t.Helper()
	want.CreatedAt, got.CreatedAt = time.Time{}, time.Time{}
	want.UpdatedAt, got.UpdatedAt = time.Time{}, time.Time{}
	want.DeletedAt, got.DeletedAt = time.Time{}, time.Time{}
	want.ExpiredAt, got.ExpiredAt = nil, nil

	assert.EqualValues(t, want, got)
}

func assertEqualOperation(t *testing.T, want interface{}, got internal.InstanceWithOperation) {
	t.Helper()
	switch want := want.(type) {
	case internal.ProvisioningOperation:
		assert.EqualValues(t, internal.OperationTypeProvision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	case internal.DeprovisioningOperation:
		assert.EqualValues(t, internal.OperationTypeDeprovision, got.Type.String)
		assert.EqualValues(t, want.State, got.State.String)
		assert.EqualValues(t, want.Description, got.Description.String)
	}
}

type instanceData struct {
	val             string
	globalAccountID string
	subAccountID    string
	expired         bool
	trial           bool
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

	instance := fixture.FixInstance(testData.val)
	instance.GlobalAccountID = gaid
	instance.SubscriptionGlobalAccountID = gaid
	instance.SubAccountID = suid
	if testData.trial {
		instance.ServicePlanID = broker.TrialPlanID
		instance.ServicePlanName = broker.TrialPlanName
	} else {
		instance.ServiceID = testData.val
		instance.ServiceName = testData.val
	}
	instance.ServicePlanName = testData.val
	instance.DashboardURL = fmt.Sprintf("https://console.%s.kyma.local", testData.val)
	instance.ProviderRegion = testData.val
	instance.Parameters.ErsContext.SubAccountID = suid
	instance.Parameters.ErsContext.GlobalAccountID = gaid
	instance.InstanceDetails = internal.InstanceDetails{}
	if testData.expired {
		instance.ExpiredAt = ptr.Time(time.Now().Add(-10 * time.Hour))
	}

	return &instance
}

func fixRuntimeOperation(operationId string) orchestration.RuntimeOperation {
	runtime := fixture.FixRuntime("runtime-id")
	runtimeOperation := fixture.FixRuntimeOperation(operationId)
	runtimeOperation.Runtime = runtime

	return runtimeOperation
}

func fixProvisionOperation(instanceId string) internal.ProvisioningOperation {
	operationId := fmt.Sprintf("%s-%d", instanceId, rand.Int())
	return fixture.FixProvisioningOperation(operationId, instanceId)

}
func fixDeprovisionOperation(instanceId string) internal.DeprovisioningOperation {
	operationId := fmt.Sprintf("%s-%d", instanceId, rand.Int())
	return fixture.FixDeprovisioningOperation(operationId, instanceId)
}

func fixUpgradeKymaOperation(testData string) internal.UpgradeKymaOperation {
	operationId := fmt.Sprintf("%s-%d", testData, rand.Int())
	return fixture.FixUpgradeKymaOperation(operationId, testData)
}
