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
	if eventType != "MASTER_SUBACCOUNT_DELETION" {
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
				{"id":145366,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906863271","eventData":{"globalAccountGuid":"22cb7ffd-6f53-4b94-9915-b8a6dc6038f3","subaccountGuid":"%s","platformID":"8e8b4231-0224-441e-a017-234cdccd816e","subdomain":"procurepartobedeleted","region":"eu10-canary"}},
				{"id":145365,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906758247","eventData":{"globalAccountGuid":"22cb7ffd-6f53-4b94-9915-b8a6dc6038f3","subaccountGuid":"%s","platformID":"064adad0-6d0c-4afb-8439-6095d86d4dfd","subdomain":"procurepardevc4","region":"eu10-canary"}},
				{"id":145364,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906403195","eventData":{"globalAccountGuid":"423e5e64-1b67-474b-87ae-8c84070cdcaa","subaccountGuid":"%s","platformID":"4ae3bbb3-dc96-47b5-81f9-66b5fa030b11","subdomain":"iotae-hotfixtmtest01","region":"eu10-canary"}}
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
