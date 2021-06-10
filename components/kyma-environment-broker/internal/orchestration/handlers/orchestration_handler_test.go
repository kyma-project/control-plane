package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestStatusHandler_AttachRoutes(t *testing.T) {
	fixID := "id-1"
	t.Run("orchestrations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID})
		require.NoError(t, err)
		err = db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: "id-2"})
		require.NoError(t, err)

		logs := logrus.New()
		kymaHandler := NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, logs)

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
		err = db.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              fixID,
				InstanceID:      fixID,
				OrchestrationID: fixID,
				State:           domain.Succeeded,
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: "4deee563-e5ec-4731-b9b1-53b42d855f0c",
				},
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID: fixID,
			},
		})
		err = db.Operations().InsertProvisioningOperation(internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         "id-2",
				InstanceID: fixID,
			},
		})
		require.NoError(t, err)

		dto := orchestration.StatusResponse{}

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		err = json.Unmarshal(rr.Body.Bytes(), &dto)
		require.NoError(t, err)
		assert.Equal(t, dto.OrchestrationID, fixID)
		assert.Len(t, dto.OperationStats, 6)
		assert.Equal(t, 1, dto.OperationStats[orchestration.Succeeded])
	})

	t.Run("kyma upgrade operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		secondID := "id-2"

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID, Type: orchestration.UpgradeKymaOrchestration})
		require.NoError(t, err)
		err = db.Operations().InsertUpgradeKymaOperation(internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              fixID,
				InstanceID:      fixID,
				OrchestrationID: fixID,
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: "4deee563-e5ec-4731-b9b1-53b42d855f0c",
				},
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID: fixID,
			},
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
		kymaHandler := NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, logs)

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

	t.Run("cluster upgrade operations", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID, Type: orchestration.UpgradeClusterOrchestration})
		require.NoError(t, err)
		err = db.Operations().InsertUpgradeClusterOperation(internal.UpgradeClusterOperation{
			Operation: internal.Operation{
				ID:              fixID,
				InstanceID:      fixID,
				OrchestrationID: fixID,
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: "4deee563-e5ec-4731-b9b1-53b42d855f0c",
				},
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID: fixID,
			},
		})
		require.NoError(t, err)

		err = db.RuntimeStates().Insert(internal.RuntimeState{ID: fixID, OperationID: fixID})
		require.NoError(t, err)

		logs := logrus.New()
		kymaHandler := NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, logs)

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

	t.Run("cancel orchestration", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()

		err := db.Orchestrations().Insert(internal.Orchestration{OrchestrationID: fixID, State: orchestration.InProgress})
		require.NoError(t, err)

		logs := logrus.New()
		kymaHandler := NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, logs)

		req, err := http.NewRequest("PUT", fmt.Sprintf("/orchestrations/%s/cancel", fixID), nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var out orchestration.UpgradeResponse

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.Equal(t, out.OrchestrationID, fixID)

		o, err := db.Orchestrations().GetByID(fixID)
		require.NoError(t, err)
		assert.Equal(t, orchestration.Canceling, o.State)
	})
}
