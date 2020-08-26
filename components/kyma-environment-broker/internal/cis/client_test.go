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
	subAccountTest1 = "fda14cab-bacc-4d0b-a10f-18557a6d9060"
	subAccountTest2 = "7514cf27-41b0-4266-a273-637cb3a2c051"
	subAccountTest3 = "47af15c8-adfe-4404-8675-525a878c4601"
)

func TestClient_FetchSubAccountsToDelete(t *testing.T) {
	t.Run("client fetched all subaccount IDs to delete", func(t *testing.T) {
		// Given
		testServer := fixHTTPServer(newServer(t))
		defer testServer.Close()

		client := NewClient(context.TODO(), Config{
			EventServiceURL: testServer.URL,
			PageSize:        "3",
		}, logger.NewLogDummy())
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubAccountsToDelete()

		// Then
		require.NoError(t, err)
		require.Len(t, saList, 3)
		require.ElementsMatch(t, saList, []string{subAccountTest1, subAccountTest2, subAccountTest3})
	})

	t.Run("error occur during fetch subaccount IDs", func(t *testing.T) {
		// Given
		srv := newServer(t)
		srv.serverErr = true
		testServer := fixHTTPServer(srv)
		defer testServer.Close()

		client := NewClient(context.TODO(), Config{
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

type server struct {
	serverErr bool
	t         *testing.T
}

func newServer(t *testing.T) *server {
	return &server{
		t: t,
	}
}

func fixHTTPServer(srv *server) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/events/v1/events/central", srv.returnCISEvents).Methods(http.MethodGet)

	return httptest.NewServer(r)
}

func (s *server) returnCISEvents(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("eventType")
	if eventType != "Subaccount_Deletion" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.serverErr {
		s.writeResponse(w, []byte(`{bad}`))
		return
	}

	pageNum := r.URL.Query().Get("pageNum")
	var response string
	if pageNum != "0" {
		response = `{}`
	} else {
		response = fmt.Sprintf(`{
			"total": 3,
			"totalPages": 1,
			"pageNum": 0,
			"morePages": "false",
			"events": [
				{
					"id": 631087,
					"actionTime": 1597135762286,
					"creationTime": 1597135763081,
					"details": {
						"description": "Subaccount deleted.",
						"guid": "%s",
						"parentGuid": "a6c5f1b0-9713-45fc-a831-ed0057a7925c",
						"displayName": "trial",
						"subaccountDescription": null,
						"region": "eu10-canary",
						"jobLocation": null,
						"subdomain": "e8b84ae5trial",
						"betaEnabled": false,
						"expiryDate": null
					},
					"globalAccountGUID": "a6c5f1b0-9713-45fc-a831-ed0057a7925c",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				},
				{
					"id": 629225,
					"actionTime": 1597090087820,
					"creationTime": 1597090088405,
					"details": {
					"description": "Subaccount deleted.",
						"guid": "%s",
						"parentGuid": "ec0a066a-60a1-4d31-b329-80cf97292789",
						"displayName": "Vered-Neo1",
						"subaccountDescription": null,
						"region": "eu1-canary",
						"jobLocation": null,
						"subdomain": "74eb3e9f-d8f5-4dc9-b2fe-5a5c061487c2",
						"betaEnabled": false,
						"expiryDate": null
					},
					"globalAccountGUID": "ec0a066a-60a1-4d31-b329-80cf97292789",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				},
				{
					"id": 629224,
					"actionTime": 1597090066116,
					"creationTime": 1597090067309,
					"details": {
					"description": "Subaccount deleted.",
						"guid": "%s",
						"parentGuid": "ec0a066a-60a1-4d31-b329-80cf97292789",
						"displayName": "anatneo",
						"subaccountDescription": null,
						"region": "eu1-canary",
						"jobLocation": null,
						"subdomain": "095db937-725d-4ce6-b802-ce33403e90d1",
						"betaEnabled": false,
						"expiryDate": null
					},
					"globalAccountGUID": "ec0a066a-60a1-4d31-b329-80cf97292789",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				}]
		}`, subAccountTest1, subAccountTest1, subAccountTest2, subAccountTest2, subAccountTest3, subAccountTest3)
	}

	s.writeResponse(w, []byte(response))
}

func (s *server) writeResponse(w http.ResponseWriter, response []byte) {
	_, err := w.Write(response)
	if err != nil {
		s.t.Errorf("fakeCisServer cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
