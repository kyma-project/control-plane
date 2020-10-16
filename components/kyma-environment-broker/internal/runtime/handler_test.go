package runtime_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	pkg "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/automock"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		testInstance1 := fixInstance(testID1, testTime1)
		testInstance2 := fixInstance(testID2, testTime2)
		testDTO1 := pkg.RuntimeDTO{
			InstanceID: testID1,
		}
		testDTO2 := pkg.RuntimeDTO{
			InstanceID: testID2,
		}

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		converter := automock.Converter{}
		converter.On("InstancesAndOperationsToDTO", testInstance1, mock.Anything, mock.Anything, mock.Anything).Return(testDTO1, nil)
		converter.On("InstancesAndOperationsToDTO", testInstance2, mock.Anything, mock.Anything, mock.Anything).Return(testDTO2, nil)
		runtimeHandler := runtime.NewHandler(instances, operations, 2, &converter)

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

		runtimeHandler := runtime.NewHandler(instances, operations, 2, runtime.NewConverter("region"))

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
		testDTO1 := pkg.RuntimeDTO{
			InstanceID: testID1,
		}
		testDTO2 := pkg.RuntimeDTO{
			InstanceID: testID2,
		}

		err := instances.Insert(testInstance1)
		require.NoError(t, err)
		err = instances.Insert(testInstance2)
		require.NoError(t, err)

		converter := automock.Converter{}
		converter.On("InstancesAndOperationsToDTO", testInstance1, mock.Anything, mock.Anything, mock.Anything).Return(testDTO1, nil)
		converter.On("InstancesAndOperationsToDTO", testInstance2, mock.Anything, mock.Anything, mock.Anything).Return(testDTO2, nil)
		runtimeHandler := runtime.NewHandler(instances, operations, 2, &converter)

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
	}
}
