package runtime_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestRuntimeHandler(t *testing.T) {
	t.Run("test pagination should work", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testID1 := "Test1"
		testID2 := "Test2"
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := internal.Instance{
			InstanceID: testID1,
			CreatedAt:  testTime1,
			Parameters: internal.ProvisioningParameters{},
		}
		testInstance2 := internal.Instance{
			InstanceID: testID2,
			CreatedAt:  testTime2,
			Parameters: internal.ProvisioningParameters{},
		}

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		req, err := http.NewRequest("GET", "/runtimes?page_size=1", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		// given
		urlPath := fmt.Sprintf("/runtimes?page=2&page_size=1")
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		logrus.Print(out.Data)
		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID2, out.Data[0].InstanceID)

	})

	t.Run("test validation should work", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "region")

		req, err := http.NewRequest("GET", "/runtimes?page_size=a", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		req, err = http.NewRequest("GET", "/runtimes?page_size=1,2,3", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)

		req, err = http.NewRequest("GET", "/runtimes?page_size=abc", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("test filtering should work", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testID1 := "Test1"
		testID2 := "Test2"
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := fixInstance(testID1, testTime1)
		testInstance2 := fixInstance(testID2, testTime2)

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?account=%s&subaccount=%s&instance_id=%s&runtime_id=%s&region=%s&shoot=%s", testID1, testID1, testID1, testID1, testID1, testID1), nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)
	})

	t.Run("test state filtering should work", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testID1 := "Test1"
		testID2 := "Test2"
		testID3 := "Test3"
		testTime1 := time.Now()
		testTime2 := time.Now().Add(time.Minute)
		testInstance1 := fixInstance(testID1, testTime1)
		testInstance2 := fixInstance(testID2, testTime2)
		testInstance3 := fixInstance(testID3, time.Now().Add(2*time.Minute))

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)
		err = instances.Insert(testInstance3)
		require.NoError(t, err)

		provOp1 := fixture.FixProvisioningOperation(fixRandomID(), testID1)
		err = operations.InsertProvisioningOperation(provOp1)
		require.NoError(t, err)

		provOp2 := fixture.FixProvisioningOperation(fixRandomID(), testID2)
		err = operations.InsertProvisioningOperation(provOp2)
		require.NoError(t, err)
		upgOp2 := fixture.FixUpgradeKymaOperation(fixRandomID(), testID2)
		upgOp2.State = domain.Failed
		upgOp2.CreatedAt = upgOp2.CreatedAt.Add(time.Minute)
		err = operations.InsertUpgradeKymaOperation(upgOp2)
		require.NoError(t, err)

		provOp3 := fixture.FixProvisioningOperation(fixRandomID(), testID3)
		err = operations.InsertProvisioningOperation(provOp3)
		require.NoError(t, err)
		upgOp3 := fixture.FixUpgradeKymaOperation(fixRandomID(), testID3)
		upgOp3.State = domain.Failed
		upgOp3.CreatedAt = upgOp3.CreatedAt.Add(time.Minute)
		err = operations.InsertUpgradeKymaOperation(upgOp3)
		require.NoError(t, err)
		deprovOp3 := fixture.FixDeprovisioningOperation(fixRandomID(), testID3)
		deprovOp3.State = domain.Succeeded
		deprovOp3.CreatedAt = deprovOp3.CreatedAt.Add(2 * time.Minute)
		err = operations.InsertDeprovisioningOperation(deprovOp3)
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		req, err := http.NewRequest("GET", fmt.Sprintf("/runtimes?state=%s", pkg.StateSucceeded), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		// when
		rr = httptest.NewRecorder()
		req, err = http.NewRequest("GET", fmt.Sprintf("/runtimes?state=%s", pkg.StateFailed), nil)
		require.NoError(t, err)
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID2, out.Data[0].InstanceID)
	})

	t.Run("should show suspension and unsuspension operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testID1 := "Test1"
		testTime1 := time.Now()
		testInstance1 := fixInstance(testID1, testTime1)

		unsuspensionOpId := "unsuspension-op-id"
		suspensionOpId := "suspension-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         "first-provisioning-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testID1,
			},
		})
		require.NoError(t, err)
		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         unsuspensionOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testID1,
			},
		})

		require.NoError(t, err)
		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         suspensionOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testID1,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testID1, out.Data[0].InstanceID)

		unsuspensionOps := out.Data[0].Status.Unsuspension.Data
		assert.Equal(t, 1, len(unsuspensionOps))
		assert.Equal(t, unsuspensionOpId, unsuspensionOps[0].OperationID)

		suspensionOps := out.Data[0].Status.Suspension.Data
		assert.Equal(t, 1, len(suspensionOps))
		assert.Equal(t, suspensionOpId, suspensionOps[0].OperationID)
	})

	t.Run("should distinguish between provisioning & unsuspension operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testInstance1 := fixture.FixInstance("instance-1")

		provisioningOpId := "provisioning-op-id"
		unsuspensionOpId := "unsuspension-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         provisioningOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testInstance1.InstanceID,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         unsuspensionOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testInstance1.InstanceID,
			},
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testInstance1.InstanceID, out.Data[0].InstanceID)
		assert.Equal(t, provisioningOpId, out.Data[0].Status.Provisioning.OperationID)

		unsuspensionOps := out.Data[0].Status.Unsuspension.Data
		assert.Equal(t, 1, len(unsuspensionOps))
		assert.Equal(t, unsuspensionOpId, unsuspensionOps[0].OperationID)
	})

	t.Run("should distinguish between deprovisioning & suspension operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)
		testInstance1 := fixture.FixInstance("instance-1")

		suspensionOpId := "suspension-op-id"
		deprovisioningOpId := "deprovisioning-op-id"

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         suspensionOpId,
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: testInstance1.InstanceID,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         deprovisioningOpId,
				Version:    0,
				CreatedAt:  time.Now().Add(1 * time.Hour),
				UpdatedAt:  time.Now().Add(1 * time.Hour),
				InstanceID: testInstance1.InstanceID,
			},
			Temporary: false,
		})
		require.NoError(t, err)

		runtimeHandler := runtime.NewHandler(instances, operations, 2, "")

		req, err := http.NewRequest("GET", "/runtimes", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		runtimeHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out pkg.RuntimesPage

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)
		assert.Equal(t, testInstance1.InstanceID, out.Data[0].InstanceID)

		suspensionOps := out.Data[0].Status.Suspension.Data
		assert.Equal(t, 1, len(suspensionOps))
		assert.Equal(t, suspensionOpId, suspensionOps[0].OperationID)

		assert.Equal(t, deprovisioningOpId, out.Data[0].Status.Deprovisioning.OperationID)
	})
}

func fixInstance(id string, t time.Time) internal.Instance {
	return internal.Instance{
		InstanceID:      id,
		CreatedAt:       t,
		GlobalAccountID: id,
		SubAccountID:    id,
		RuntimeID:       id,
		ServiceID:       id,
		ServiceName:     id,
		ServicePlanID:   id,
		ServicePlanName: id,
		DashboardURL:    fmt.Sprintf("https://console.%s.kyma.local", id),
		ProviderRegion:  id,
		Parameters:      internal.ProvisioningParameters{},
	}
}

func fixRandomID() string {
	return rand.String(16)
}
