package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type FakeTokenSource string

var fixToken FakeTokenSource = "fake-token-1234"

func (t FakeTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: string(t),
		Expiry:      time.Now().Add(time.Duration(12 * time.Hour)),
	}, nil
}

var orch1 = fixStatusResponse("orchestration1")
var orch2 = fixStatusResponse("orchestration2")
var orch3 = fixStatusResponse("orchestration3")
var orch4 = fixStatusResponse("orchestration4")
var orchs = []StatusResponse{orch1, orch2, orch3, orch4}
var operations = []OperationResponse{
	fixOperationResponse("operation1", orch1.OrchestrationID),
	fixOperationResponse("operation2", orch1.OrchestrationID),
	fixOperationResponse("operation3", orch1.OrchestrationID),
	fixOperationResponse("operation4", orch1.OrchestrationID),
}

func TestClient_ListOrchestrations(t *testing.T) {
	t.Run("test_URL_params_pagination__NoError_path", func(t *testing.T) {
		// given
		called := 0
		params := ListParameters{
			PageSize: 2,
			States:   []string{"failed", "in progress"},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/orchestrations", r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))
			query := r.URL.Query()
			assert.ElementsMatch(t, []string{strconv.Itoa(called)}, query[pagination.PageParam])
			assert.ElementsMatch(t, []string{strconv.Itoa(params.PageSize)}, query[pagination.PageSizeParam])
			assert.ElementsMatch(t, params.States, query[StateParam])

			err := respondStatusList(w, orchs[(called-1)*params.PageSize:called*params.PageSize], 4)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		srl, err := client.ListOrchestrations(params)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, called)
		assert.Equal(t, 4, srl.Count)
		assert.Equal(t, 4, srl.TotalCount)
		assert.Len(t, srl.Data, 4)
		assert.Equal(t, orch1.OrchestrationID, srl.Data[0].OrchestrationID)
		assert.Equal(t, orch2.OrchestrationID, srl.Data[1].OrchestrationID)
		assert.Equal(t, orch3.OrchestrationID, srl.Data[2].OrchestrationID)
		assert.Equal(t, orch4.OrchestrationID, srl.Data[3].OrchestrationID)
	})
}

func TestClient_GetOrchestration(t *testing.T) {
	t.Run("test_URL__NoError_path", func(t *testing.T) {
		// given
		called := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, fmt.Sprintf("/orchestrations/%s", orch1.OrchestrationID), r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))

			err := respondStatus(w, orch1)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		sr, err := client.GetOrchestration(orch1.OrchestrationID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, orch1.OrchestrationID, sr.OrchestrationID)
	})
}

func TestClient_ListOperations(t *testing.T) {
	t.Run("test_URL_params_pagination__NoError_path", func(t *testing.T) {
		// given
		called := 0
		params := ListParameters{
			PageSize: 2,
			States:   []string{"failed", "in progress"},
		}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, fmt.Sprintf("/orchestrations/%s/operations", orch1.OrchestrationID), r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))
			query := r.URL.Query()
			assert.ElementsMatch(t, []string{strconv.Itoa(called)}, query[pagination.PageParam])
			assert.ElementsMatch(t, []string{strconv.Itoa(params.PageSize)}, query[pagination.PageSizeParam])
			assert.ElementsMatch(t, params.States, query[StateParam])

			err := respondOperationList(w, operations[(called-1)*params.PageSize:called*params.PageSize], 4)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		orl, err := client.ListOperations(orch1.OrchestrationID, params)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, called)
		assert.Equal(t, 4, orl.Count)
		assert.Equal(t, 4, orl.TotalCount)
		assert.Len(t, orl.Data, 4)
		for i := 0; i < 4; i++ {
			assert.Equal(t, orch1.OrchestrationID, orl.Data[i].OrchestrationID)
			assert.Equal(t, operations[i].OperationID, orl.Data[i].OperationID)
		}
	})
}

func TestClient_GetOperation(t *testing.T) {
	t.Run("test_URL__NoError_path", func(t *testing.T) {
		// given
		called := 0
		oper := fixOperationDetailResponse("operation1", orch1.OrchestrationID)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, fmt.Sprintf("/orchestrations/%s/operations/%s", orch1.OrchestrationID, oper.OperationID), r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))

			err := respondOperationDetail(w, oper)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		od, err := client.GetOperation(orch1.OrchestrationID, oper.OperationID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, orch1.OrchestrationID, od.OrchestrationID)
		assert.Equal(t, oper.OperationID, od.OperationID)
	})
}

func TestClient_UpgradeKyma(t *testing.T) {
	t.Run("test_URL_request_body_NoError_path", func(t *testing.T) {
		// given
		called := 0
		params := Parameters{
			Targets: TargetSpec{
				Include: []RuntimeTarget{
					{
						Target: TargetAll,
					},
				},
				Exclude: []RuntimeTarget{
					{
						GlobalAccount: "GA",
					},
				},
			},
			Strategy: StrategySpec{
				Type:     ParallelStrategy,
				Schedule: MaintenanceWindow,
				Parallel: ParallelStrategySpec{
					Workers: 2,
				},
			},
		}
		orchestrationID := orch1.OrchestrationID
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/upgrade/kyma", r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))
			reqBody := Parameters{}
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			require.NoError(t, err)
			assert.True(t, reflect.DeepEqual(params, reqBody))

			err = respondUpgrade(w, orchestrationID)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		ur, err := client.UpgradeKyma(params)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, orchestrationID, ur.OrchestrationID)
	})
}

func TestClient_CancelOrchestration(t *testing.T) {
	t.Run("test_URL__NoError_path", func(t *testing.T) {
		// given
		called := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, fmt.Sprintf("/orchestrations/%s/cancel", orch1.OrchestrationID), r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))

			err := respondStatus(w, orch1)
			require.NoError(t, err)
		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)

		// when
		err := client.CancelOrchestration(orch1.OrchestrationID)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
	})
}

func TestClient_RetryOrchestration(t *testing.T) {
	t.Run("test_URL_NoError_path", func(t *testing.T) {
		// given
		called := 0
		operationIDs := []string{"operation_id_0", "operation_id_1"}
		ids := []string{}
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called++
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, fmt.Sprintf("/orchestrations/%s/retry", orch1.OrchestrationID), r.URL.Path)
			assert.Equal(t, fmt.Sprintf("Bearer %s", fixToken), r.Header.Get("Authorization"))
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			for _, id := range operationIDs {
				ids = append(ids, "operation-id="+id)
			}

			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			body := buf.String()
			assert.Equal(t, strings.Join(operationIDs, "&"), body)

			err := respondRetry(w, orch1.OrchestrationID, operationIDs)
			require.NoError(t, err)

		}))
		defer ts.Close()
		client := NewClient(context.TODO(), ts.URL, fixToken)
		expectedRr := RetryResponse{
			OrchestrationID: orch1.OrchestrationID,
			RetryOperations: operationIDs,
			Msg:             "retry operations are queued for processing",
		}

		// when
		rr, err := client.RetryOrchestration(orch1.OrchestrationID, operationIDs)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, called)
		assert.Equal(t, expectedRr, rr)

		// when
		operationIDs = nil
		expectedRr.RetryOperations = []string{"operation-ID-3", "operation-ID-4"}
		rr, err = client.RetryOrchestration(orch1.OrchestrationID, operationIDs)

		// then
		require.NoError(t, err)
		assert.Equal(t, 2, called)
		assert.Equal(t, expectedRr, rr)
	})
}

func fixStatusResponse(id string) StatusResponse {
	return StatusResponse{
		OrchestrationID: id,
		State:           "succeeded",
		Description:     id,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Parameters:      Parameters{},
		OperationStats: map[string]int{
			"succeeded": 5,
		},
	}
}

func fixOperationResponse(id, orchestrationID string) OperationResponse {
	return OperationResponse{
		OperationID:     id,
		RuntimeID:       id,
		OrchestrationID: orchestrationID,
	}
}

func fixOperationDetailResponse(id, orchestrationID string) OperationDetailResponse {
	return OperationDetailResponse{
		OperationResponse: fixOperationResponse(id, orchestrationID),
	}
}

func respondStatusList(w http.ResponseWriter, statuses []StatusResponse, totalCount int) error {
	srl := StatusResponseList{
		Data:       statuses,
		Count:      len(statuses),
		TotalCount: totalCount,
	}
	data, err := json.Marshal(srl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func respondStatus(w http.ResponseWriter, status StatusResponse) error {
	data, err := json.Marshal(status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func respondOperationList(w http.ResponseWriter, operations []OperationResponse, totalCount int) error {
	orl := OperationResponseList{
		Data:       operations,
		Count:      len(operations),
		TotalCount: totalCount,
	}
	data, err := json.Marshal(orl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func respondOperationDetail(w http.ResponseWriter, operation OperationDetailResponse) error {
	data, err := json.Marshal(operation)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func respondUpgrade(w http.ResponseWriter, orchestrationID string) error {
	ur := UpgradeResponse{
		OrchestrationID: orchestrationID,
	}
	data, err := json.Marshal(ur)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, err = w.Write(data)
	return err
}

func respondRetry(w http.ResponseWriter, orchestrationID string, operationIDs []string) error {
	rr := RetryResponse{
		OrchestrationID: orchestrationID,
	}

	if len(operationIDs) == 0 {
		rr.RetryOperations = []string{"operation-ID-3", "operation-ID-4"}
		rr.Msg = "retry operations are queued for processing"
	} else {
		rr.RetryOperations = operationIDs
		rr.Msg = "retry operations are queued for processing"
	}

	data, err := json.Marshal(rr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, err = w.Write(data)
	return err
}
