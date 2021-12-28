package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
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

	t.Run("Kyma 2.0 upgrade operation", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()

		instanceID := "instanceID"
		provisioningOp1ID := "provisioningOp1ID"

		provisioningOp1 := internal.ProvisioningOperation{
			Operation: internal.Operation{
				ID:         provisioningOp1ID,
				InstanceID: instanceID,
			},
		}

		err := db.Operations().InsertProvisioningOperation(provisioningOp1)
		require.NoError(t, err)

		orchestration1ID := "ochestration1ID"
		orchestration1 := internal.Orchestration{
			OrchestrationID: orchestration1ID,
			Type:            orchestration.UpgradeKymaOrchestration,
		}

		err = db.Orchestrations().Insert(orchestration1)
		require.NoError(t, err)

		upgradeKymaOp1ID := "upgradeKymaOperation1ID"
		upgradeKymaOp1 := internal.UpgradeKymaOperation{
			Operation: internal.Operation{
				ID:              upgradeKymaOp1ID,
				InstanceID:      instanceID,
				OrchestrationID: orchestration1ID,
				ProvisioningParameters: internal.ProvisioningParameters{
					PlanID: broker.AzurePlanID,
				},
			},
			RuntimeOperation: orchestration.RuntimeOperation{
				ID: upgradeKymaOp1ID,
			},
		}

		err = db.Operations().InsertUpgradeKymaOperation(upgradeKymaOp1)
		require.NoError(t, err)

		runtimeStateWithClusterSetupID := "runtimeStateWithClusterSetupID"
		runtimeStateWithClusterSetup := internal.RuntimeState{
			ID:          runtimeStateWithClusterSetupID,
			RuntimeID:   uuid.NewString(),
			OperationID: upgradeKymaOp1ID,
			ClusterSetup: &reconcilerApi.Cluster{
				RuntimeID: uuid.NewString(),
				KymaConfig: reconcilerApi.KymaConfig{
					Version: "2.0.0",
					Profile: string(gqlschema.KymaProfileProduction),
					Components: []reconcilerApi.Component{
						{
							URL:       "component1URL.local",
							Component: "component1",
							Namespace: "test",
							Configuration: []reconcilerApi.Configuration{
								{
									Key:    "key1",
									Value:  "value1",
									Secret: false,
								},
								{
									Key:    "key2",
									Value:  "value2",
									Secret: true,
								},
							},
						},
					},
					Administrators: []string{"admin1@test.com", "admin2@test.com"},
				},
			},
		}

		err = db.RuntimeStates().Insert(runtimeStateWithClusterSetup)
		require.NoError(t, err)

		logs := logrus.New()
		kymaHandler := NewOrchestrationStatusHandler(db.Operations(), db.Orchestrations(), db.RuntimeStates(), 100, logs)

		urlPath := fmt.Sprintf("/orchestrations/%s/operations", orchestration1ID)
		req, err := http.NewRequest("GET", urlPath, nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var opResponseList orchestration.OperationResponseList

		err = json.Unmarshal(rr.Body.Bytes(), &opResponseList)
		require.NoError(t, err)

		assert.Len(t, opResponseList.Data, 1)
		assert.Equal(t, 1, opResponseList.TotalCount)
		assert.Equal(t, 1, opResponseList.Count)

		// given
		urlPath = fmt.Sprintf("/orchestrations/%s/operations/%s", orchestration1ID, upgradeKymaOp1ID)
		req, err = http.NewRequest(http.MethodGet, urlPath, nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusOK, rr.Code)

		var opDetailResponse orchestration.OperationDetailResponse
		err = json.Unmarshal(rr.Body.Bytes(), &opDetailResponse)
		require.NoError(t, err)

		expectedKymaConfig := gqlschema.KymaConfigInput{
			Version: "2.0.0",
			Profile: (*gqlschema.KymaProfile)(ptr.String("Production")),
			Components: []*gqlschema.ComponentConfigurationInput{
				{
					Component: "component1",
					Namespace: "test",
					SourceURL: ptr.String("component1URL.local"),
					Configuration: []*gqlschema.ConfigEntryInput{
						{
							Key:    "key1",
							Value:  "value1",
							Secret: ptr.Bool(false),
						},
						{
							Key:    "key2",
							Value:  "value2",
							Secret: ptr.Bool(true),
						},
					},
				},
			},
		}

		assert.Equal(t, opDetailResponse.OrchestrationID, orchestration1ID)
		assert.Equal(t, opDetailResponse.OperationID, upgradeKymaOp1ID)
		assert.NotNil(t, opDetailResponse.KymaConfig)
		assertKymaConfigValues(t, expectedKymaConfig, *opDetailResponse.KymaConfig)
	})
}

func assertKymaConfigValues(t *testing.T, expected, actual gqlschema.KymaConfigInput) {
	assert.Equal(t, expected.Version, actual.Version)
	assert.Equal(t, *expected.Profile, *actual.Profile)
	if len(expected.Components) > 0 {
		for i, cmp := range expected.Components {
			if len(cmp.Configuration) > 0 {
				for j, cfg := range cmp.Configuration {
					assert.Equal(t, cfg.Value, actual.Components[i].Configuration[j].Value)
					assert.Equal(t, cfg.Key, actual.Components[i].Configuration[j].Key)
					assert.Equal(t, *cfg.Secret, *actual.Components[i].Configuration[j].Secret)
				}
			}
			assert.Equal(t, cmp.Component, actual.Components[i].Component)
			assert.Equal(t, cmp.Namespace, actual.Components[i].Namespace)
			if cmp.SourceURL != nil {
				assert.Equal(t, *cmp.SourceURL, *actual.Components[i].SourceURL)
			}
		}
	}
}
