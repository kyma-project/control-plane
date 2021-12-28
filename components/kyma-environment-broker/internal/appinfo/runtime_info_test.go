package appinfo_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/appinfo/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// go test ./internal/appinfo -update -run=TestRuntimeInfoHandlerSuccess
func TestRuntimeInfoHandlerSuccess(t *testing.T) {
	tests := map[string]struct {
		instances     []internal.Instance
		provisionOp   []internal.ProvisioningOperation
		deprovisionOp []internal.DeprovisioningOperation
	}{
		"no instances": {
			instances: []internal.Instance{},
		},
		"instances without operations": {
			instances: []internal.Instance{
				fixInstance(1), fixInstance(2), fixInstance(2),
			},
		},
		"instances without service and plan name should have defaults": {
			instances: func() []internal.Instance {
				i := fixInstance(1)
				i.ServicePlanName = ""
				i.ServiceName = ""
				// selecting servicePlanName based on existing real planID
				i.ServicePlanID = broker.GCPPlanID
				return []internal.Instance{i}
			}(),
		},
		"instances without platform region name should have default": {
			instances: func() []internal.Instance {
				i := fixInstance(1)
				// the platform_region is not specified
				i.Parameters = internal.ProvisioningParameters{}
				return []internal.Instance{i}
			}(),
		},
		"instances with provision operation": {
			instances: []internal.Instance{
				fixInstance(1), fixInstance(2), fixInstance(3),
			},
			provisionOp: []internal.ProvisioningOperation{
				fixProvisionOperation(1), fixProvisionOperation(2),
			},
		},
		"instances with deprovision operation": {
			instances: []internal.Instance{
				fixInstance(1), fixInstance(2), fixInstance(3),
			},
			deprovisionOp: []internal.DeprovisioningOperation{
				fixDeprovisionOperation(1), fixDeprovisionOperation(2),
			},
		},
		"instances with provision and deprovision operations": {
			instances: []internal.Instance{
				fixInstance(1), fixInstance(2), fixInstance(3),
			},
			provisionOp: []internal.ProvisioningOperation{
				fixProvisionOperation(1), fixProvisionOperation(2),
			},
			deprovisionOp: []internal.DeprovisioningOperation{
				fixDeprovisionOperation(1), fixDeprovisionOperation(2),
			},
		},
	}
	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			// given
			var (
				fixReq     = httptest.NewRequest("GET", "http://example.com/foo", nil)
				respSpy    = httptest.NewRecorder()
				writer     = httputil.NewResponseWriter(logger.NewLogDummy(), true)
				memStorage = newInMemoryStorage(t, tc.instances, tc.provisionOp, tc.deprovisionOp)
			)

			handler := appinfo.NewRuntimeInfoHandler(memStorage.Instances(), memStorage.Operations(), broker.PlansConfig{}, "default-region", writer)

			// when
			handler.ServeHTTP(respSpy, fixReq)

			// then
			assert.Equal(t, http.StatusOK, respSpy.Result().StatusCode)
			assert.Equal(t, "application/json", respSpy.Result().Header.Get("Content-Type"))

			assertJSONWithGoldenFile(t, respSpy.Body.Bytes())
		})
	}
}

func TestRuntimeInfoHandlerFailures(t *testing.T) {
	// given
	var (
		fixReq  = httptest.NewRequest("GET", "http://example.com/foo", nil)
		respSpy = httptest.NewRecorder()
		writer  = httputil.NewResponseWriter(logger.NewLogDummy(), true)
		expBody = `{
				  "status": 500,
				  "requestId": "",
				  "message": "Something went very wrong. Please try again.",
				  "details": "while fetching all instances: ups.. internal info"
				}`
	)

	storageMock := &automock.InstanceFinder{}
	defer storageMock.AssertExpectations(t)
	storageMock.On("FindAllJoinedWithOperations", mock.Anything).Return(nil, errors.New("ups.. internal info"))
	handler := appinfo.NewRuntimeInfoHandler(storageMock, nil, broker.PlansConfig{}, "", writer)

	// when
	handler.ServeHTTP(respSpy, fixReq)

	// then
	assert.Equal(t, http.StatusInternalServerError, respSpy.Result().StatusCode)
	assert.Equal(t, "application/json", respSpy.Result().Header.Get("Content-Type"))

	assert.JSONEq(t, expBody, respSpy.Body.String())
}

func TestRuntimeInfoHandlerOperationRecognition(t *testing.T) {
	t.Run("should distinguish between provisioning & unsuspension operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)

		testInstance1 := fixture.FixInstance("instance-1")
		testInstance2 := fixture.FixInstance("instance-2")

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		provisioningOpId1 := "provisioning-op-1"
		provisioningOpId2 := "provisioning-op-2"
		unsuspensionOpId1 := "unsuspension-op-1"
		unsuspensionOpId2 := "unsuspension-op-2"
		provisioningOpDesc1 := "succeeded provisioning operation 1"
		provisioningOpDesc2 := "succeeded provisioning operation 2"
		unsuspensionOpDesc1 := "succeeded unsuspension operation 1"
		unsuspensionOpDesc2 := "succeeded unsuspension operation 2"

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          provisioningOpId1,
				Version:     0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: provisioningOpDesc1,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          unsuspensionOpId1,
				Version:     0,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now().Add(1 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: unsuspensionOpDesc1,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          unsuspensionOpId2,
				Version:     0,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now().Add(1 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance2.InstanceID,
				State:       domain.Succeeded,
				Description: unsuspensionOpDesc2,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          provisioningOpId2,
				Version:     0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance2.InstanceID,
				State:       domain.Succeeded,
				Description: provisioningOpDesc2,
			},
		})
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "/info/runtimes", nil)
		require.NoError(t, err)

		responseWriter := httputil.NewResponseWriter(logger.NewLogDummy(), true)
		runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(instances, operations, broker.PlansConfig{}, "", responseWriter)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.Handle("/info/runtimes", runtimesInfoHandler)

		// when
		runtimesInfoHandler.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out []*appinfo.RuntimeDTO

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 2, len(out))
		assert.Equal(t, testInstance1.InstanceID, out[0].ServiceInstanceID)
		assert.Equal(t, testInstance2.InstanceID, out[1].ServiceInstanceID)
		assert.Equal(t, provisioningOpDesc1, out[0].Status.Provisioning.Description)
		assert.Equal(t, provisioningOpDesc2, out[1].Status.Provisioning.Description)

	})

	t.Run("should distinguish between deprovisioning & suspension operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)

		testInstance1 := fixture.FixInstance("instance-1")
		testInstance2 := fixture.FixInstance("instance-2")

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		deprovisioningOpId1 := "deprovisioning-op-1"
		deprovisioningOpId2 := "deprovisioning-op-2"
		suspensionOpId1 := "suspension-op-1"
		suspensionOpId2 := "suspension-op-2"
		deprovisioningOpDesc1 := "succeeded deprovisioning operation 1"
		deprovisioningOpDesc2 := "succeeded deprovisioning operation 2"
		suspensionOpDesc1 := "succeeded suspension operation 1"
		suspensionOpDesc2 := "succeeded suspension operation 2"

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          suspensionOpId1,
				Version:     0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: suspensionOpDesc1,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          deprovisioningOpId1,
				Version:     0,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now().Add(1 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: deprovisioningOpDesc1,
			},
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          deprovisioningOpId2,
				Version:     0,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now().Add(1 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance2.InstanceID,
				State:       domain.Succeeded,
				Description: deprovisioningOpDesc2,
			},
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          suspensionOpId2,
				Version:     0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance2.InstanceID,
				State:       domain.Succeeded,
				Description: suspensionOpDesc2,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "/info/runtimes", nil)
		require.NoError(t, err)

		responseWriter := httputil.NewResponseWriter(logger.NewLogDummy(), true)
		runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(instances, operations, broker.PlansConfig{}, "", responseWriter)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.Handle("/info/runtimes", runtimesInfoHandler)

		// when
		runtimesInfoHandler.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out []*appinfo.RuntimeDTO

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 2, len(out))
		assert.Equal(t, testInstance1.InstanceID, out[0].ServiceInstanceID)
		assert.Equal(t, testInstance2.InstanceID, out[1].ServiceInstanceID)
		assert.Equal(t, deprovisioningOpDesc1, out[0].Status.Deprovisioning.Description)
		assert.Equal(t, deprovisioningOpDesc2, out[1].Status.Deprovisioning.Description)

	})

	t.Run("should recognize prov & deprov ops among suspend/unsuspend operations", func(t *testing.T) {
		// given
		operations := memory.NewOperation()
		instances := memory.NewInstance(operations)

		testInstance1 := fixture.FixInstance("instance-1")

		err := instances.Insert(testInstance1)
		require.NoError(t, err)

		provisioningOpId := "provisioning-op"
		deprovisioningOpId := "deprovisioning-op"
		suspensionOpId1 := "suspension-op-1"
		suspensionOpId2 := "suspension-op-2"
		unsuspensionOpId1 := "unsuspension-op-1"
		unsuspensionOpId2 := "unsuspension-op-2"
		provisioningOpDesc := "succeeded provisioning operation"
		deprovisioningOpDesc := "succeeded deprovisioning operation"
		suspensionOpDesc1 := "failed suspension operation 1"
		suspensionOpDesc2 := "succeeded suspension operation 2"
		unsuspensionOpDesc1 := "failed unsuspension operation 1"
		unsuspensionOpDesc2 := "succeeded unsuspension operation 2"

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          provisioningOpId,
				Version:     0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now().Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: provisioningOpDesc,
			},
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          suspensionOpId1,
				Version:     0,
				CreatedAt:   time.Now().Add(1 * time.Hour),
				UpdatedAt:   time.Now().Add(1 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Failed,
				Description: suspensionOpDesc1,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          suspensionOpId2,
				Version:     0,
				CreatedAt:   time.Now().Add(2 * time.Hour),
				UpdatedAt:   time.Now().Add(2 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: suspensionOpDesc2,
			},
			Temporary: true,
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          unsuspensionOpId1,
				Version:     0,
				CreatedAt:   time.Now().Add(3 * time.Hour),
				UpdatedAt:   time.Now().Add(3 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Failed,
				Description: unsuspensionOpDesc1,
			},
		})
		require.NoError(t, err)

		err = operations.InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:          unsuspensionOpId2,
				Version:     0,
				CreatedAt:   time.Now().Add(4 * time.Hour),
				UpdatedAt:   time.Now().Add(4 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeProvision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: unsuspensionOpDesc2,
			},
		})
		require.NoError(t, err)

		err = operations.InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:          deprovisioningOpId,
				Version:     0,
				CreatedAt:   time.Now().Add(5 * time.Hour),
				UpdatedAt:   time.Now().Add(5 * time.Hour).Add(5 * time.Minute),
				Type:        internal.OperationTypeDeprovision,
				InstanceID:  testInstance1.InstanceID,
				State:       domain.Succeeded,
				Description: deprovisioningOpDesc,
			},
		})
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "/info/runtimes", nil)
		require.NoError(t, err)

		responseWriter := httputil.NewResponseWriter(logger.NewLogDummy(), true)
		runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(instances, operations, broker.PlansConfig{}, "", responseWriter)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.Handle("/info/runtimes", runtimesInfoHandler)

		// when
		runtimesInfoHandler.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out []*appinfo.RuntimeDTO

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, 1, len(out))
		assert.Equal(t, testInstance1.InstanceID, out[0].ServiceInstanceID)
		assert.Equal(t, provisioningOpDesc, out[0].Status.Provisioning.Description)
		assert.Equal(t, deprovisioningOpDesc, out[0].Status.Deprovisioning.Description)

	})
}

func assertJSONWithGoldenFile(t *testing.T, gotRawJSON []byte) {
	t.Helper()
	g := goldie.New(t, goldie.WithNameSuffix(".golden.json"))

	var jsonGoType interface{}
	require.NoError(t, json.Unmarshal(gotRawJSON, &jsonGoType))
	g.AssertJson(t, t.Name(), jsonGoType)
}

func fixTime() time.Time {
	return time.Date(2020, 04, 21, 0, 0, 23, 42, time.UTC)
}

func fixInstance(idx int) internal.Instance {
	return internal.Instance{
		InstanceID:      fmt.Sprintf("InstanceID field. IDX: %d", idx),
		RuntimeID:       fmt.Sprintf("RuntimeID field. IDX: %d", idx),
		GlobalAccountID: fmt.Sprintf("GlobalAccountID field. IDX: %d", idx),
		SubAccountID:    fmt.Sprintf("SubAccountID field. IDX: %d", idx),
		ServiceID:       fmt.Sprintf("ServiceID field. IDX: %d", idx),
		ServiceName:     fmt.Sprintf("ServiceName field. IDX: %d", idx),
		ServicePlanID:   fmt.Sprintf("ServicePlanID field. IDX: %d", idx),
		ServicePlanName: fmt.Sprintf("ServicePlanName field. IDX: %d", idx),
		DashboardURL:    fmt.Sprintf("DashboardURL field. IDX: %d", idx),
		Parameters: internal.ProvisioningParameters{
			PlatformRegion: fmt.Sprintf("region-value-idx-%d", idx),
		},
		CreatedAt: fixTime().Add(time.Duration(idx) * time.Second),
		UpdatedAt: fixTime().Add(time.Duration(idx) * time.Minute),
		DeletedAt: fixTime().Add(time.Duration(idx) * time.Hour),
	}
}

func newInMemoryStorage(t *testing.T,
	instances []internal.Instance,
	provisionOp []internal.ProvisioningOperation,
	deprovisionOp []internal.DeprovisioningOperation) storage.BrokerStorage {

	t.Helper()
	memStorage := storage.NewMemoryStorage()
	for _, i := range instances {
		require.NoError(t, memStorage.Instances().Insert(i))
	}
	for _, op := range provisionOp {
		require.NoError(t, memStorage.Operations().InsertProvisioningOperation(op))
	}
	for _, op := range deprovisionOp {
		require.NoError(t, memStorage.Operations().InsertDeprovisioningOperation(op))
	}

	return memStorage
}

func fixProvisionOperation(idx int) internal.ProvisioningOperation {
	return internal.ProvisioningOperation{
		Operation: fixSucceededOperation(idx),
	}
}
func fixDeprovisionOperation(idx int) internal.DeprovisioningOperation {
	return internal.DeprovisioningOperation{
		Operation: fixSucceededOperation(idx),
	}
}

func fixSucceededOperation(idx int) internal.Operation {
	return internal.Operation{
		ID:                     fmt.Sprintf("Operation ID field. IDX: %d", idx),
		Version:                0,
		CreatedAt:              fixTime().Add(time.Duration(idx) * 24 * time.Hour),
		UpdatedAt:              fixTime().Add(time.Duration(idx) * 48 * time.Hour),
		InstanceID:             fmt.Sprintf("InstanceID field. IDX: %d", idx),
		ProvisionerOperationID: fmt.Sprintf("ProvisionerOperationID field. IDX: %d", idx),
		State:                  domain.Succeeded,
		Description:            fmt.Sprintf("esc for succeeded op.. IDX: %d", idx),
	}
}
