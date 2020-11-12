package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestKymaOrchestrationHandler_(t *testing.T) {
	fixID := "id-1"

	t.Run("upgrade", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		logs := logrus.New()
		q := process.NewQueue(&testExecutor{}, logs)
		kymaHandler := handlers.NewKymaOrchestrationHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, q, logs)

		params := orchestration.Parameters{
			Targets: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						RuntimeID: "test",
					},
				},
			},
		}
		p, err := json.Marshal(&params)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/upgrade/kyma", bytes.NewBuffer(p))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusAccepted, rr.Code)

		var out orchestration.UpgradeResponse

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.NotEmpty(t, out.OrchestrationID)

		// check via GET endpoint
		urlPath := fmt.Sprintf("/orchestrations/%s", out.OrchestrationID)
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		dto := orchestration.StatusResponse{}

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &dto)
		require.NoError(t, err)
		assert.Equal(t, dto.OrchestrationID, out.OrchestrationID)
		assert.Equal(t, dto.Parameters.Strategy.Type, orchestration.ParallelStrategy)
		assert.Equal(t, dto.Parameters.Strategy.Schedule, orchestration.Immediate)
	})

	t.Run("orchestrations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID})
		require.NoError(t, err)
		err = db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: "id-2"})
		require.NoError(t, err)

		logs := logrus.New()
		q := process.NewQueue(&testExecutor{}, logs)
		kymaHandler := handlers.NewKymaOrchestrationHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, q, logs)

		req, err := http.NewRequest("GET", "/orchestrations?page_size=1", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out orchestration.StatusResponseList

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Len(t, out.Data, 1)
		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)

		// given
		urlPath := fmt.Sprintf("/orchestrations?page=2&page_size=1")
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Equal(t, 2, out.TotalCount)
		assert.Equal(t, 1, out.Count)

		// given
		urlPath = fmt.Sprintf("/orchestrations/%s", fixID)
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		dto := orchestration.StatusResponse{}

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &dto)
		require.NoError(t, err)
		assert.Equal(t, dto.OrchestrationID, fixID)
	})

	t.Run("operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		secondID := "id-2"

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID})
		require.NoError(t, err)
		err = db.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			RuntimeOperation: internal.RuntimeOperation{
				Operation: internal.Operation{
					ID:              fixID,
					InstanceID:      fixID,
					OrchestrationID: fixID,
				},
			},
			PlanID: "4deee563-e5ec-4731-b9b1-53b42d855f0c",
		})
		err = db.Operations().InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         secondID,
				InstanceID: fixID,
			},
		})
		require.NoError(t, err)

		err = db.RuntimeStates().Insert(internal.RuntimeState{ID: secondID, OperationID: secondID})
		require.NoError(t, err)
		err = db.RuntimeStates().Insert(internal.RuntimeState{ID: fixID, OperationID: fixID})
		require.NoError(t, err)

		logs := logrus.New()
		q := process.NewQueue(&testExecutor{}, logs)
		kymaHandler := handlers.NewKymaOrchestrationHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, q, logs)

		urlPath := fmt.Sprintf("/orchestrations/%s/operations", fixID)
		req, err := http.NewRequest("GET", urlPath, nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out orchestration.OperationResponseList

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Len(t, out.Data, 1)
		assert.Equal(t, 1, out.TotalCount)
		assert.Equal(t, 1, out.Count)

		// given
		urlPath = fmt.Sprintf("/orchestrations/%s/operations/%s", fixID, fixID)
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)
		rr = httptest.NewRecorder()

		dto := orchestration.OperationDetailResponse{}

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &dto)
		require.NoError(t, err)
		assert.Equal(t, dto.OrchestrationID, fixID)
		assert.Equal(t, dto.OperationID, fixID)
	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
