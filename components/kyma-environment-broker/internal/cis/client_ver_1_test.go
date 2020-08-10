package cis

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

const (
	subAccountTest4 = "c3db1e8e-1b7b-4122-8966-86f7eb3e3b6f"
	subAccountTest5 = "5d57aa1c-00a4-45f6-a56e-9e51ce0c5c2f"
	subAccountTest6 = "9f5e340f-c7f7-4df6-bb55-c3ed6b80945a"
)

func TestClientVer1_FetchSubAccountsToDelete(t *testing.T) {
	t.Run("client fetched all subaccount IDs to delete", func(t *testing.T) {
		// Given
		testServer := fixHTTPServerVer1(newServerVer1(t))
		defer testServer.Close()

		client := NewClientVer1(context.TODO(), Config{
			EventServiceURL: testServer.URL,
			PageSize:        "3",
		}, logger.NewLogDummy())
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubAccountsToDelete()

		// Then
		require.NoError(t, err)
		require.Len(t, saList, 3)
		require.ElementsMatch(t, saList, []string{subAccountTest4, subAccountTest5, subAccountTest6})
	})

	t.Run("error occur during fetch subaccount IDs", func(t *testing.T) {
		// Given
		srv := newServerVer1(t)
		srv.serverErr = true
		testServer := fixHTTPServerVer1(srv)
		defer testServer.Close()

		client := NewClientVer1(context.TODO(), Config{
			EventServiceURL: testServer.URL,
			PageSize:        "3",
		}, logger.NewLogDummy())
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubAccountsToDelete()

		// Then
		require.Error(t, err)
		require.Len(t, saList, 0)
	})
}

type serverVer1 struct {
	serverErr bool
	t         *testing.T
}

func newServerVer1(t *testing.T) *serverVer1 {
	return &serverVer1{
		t: t,
	}
}

func fixHTTPServerVer1(srv *serverVer1) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/public/rest/v2/events", srv.returnCISEvents).Methods(http.MethodGet)

	return httptest.NewServer(r)
}

func (s *serverVer1) returnCISEvents(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("type")
	if eventType != "SUBACCOUNT_DELETION" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.serverErr {
		s.writeResponse(w, []byte(`{bad}`))
		return
	}

	pageNum := r.URL.Query().Get("page")
	var response string
	if pageNum != "1" {
		response = `{}`
	} else {
		response = fmt.Sprintf(`{
			"events":[
				{"id":1719032,"type":"SUBACCOUNT_DELETION","timestamp":"1597282739858","eventData":"{\"name\":\"yffed5901\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597282692650}"},
				{"id":1719030,"type":"SUBACCOUNT_DELETION","timestamp":"1597280624603","eventData":"{\"name\":\"y66e408bb\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597280564719}"},
				{"id":1719028,"type":"SUBACCOUNT_DELETION","timestamp":"1597276747286","eventData":"{\"name\":\"y11e3382d\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276634965}"}
			],
			"totalResults":3,
			"totalPages":1
		}`, subAccountTest4, subAccountTest5, subAccountTest6)
	}

	s.writeResponse(w, []byte(response))
}

func (s *serverVer1) writeResponse(w http.ResponseWriter, response []byte) {
	_, err := w.Write(response)
	if err != nil {
		s.t.Errorf("fakeCisServer cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
