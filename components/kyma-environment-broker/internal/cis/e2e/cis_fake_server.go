package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type server struct {
	t *testing.T
}

func newServer(t *testing.T) *server {
	return &server{
		t: t,
	}
}

func fixHTTPServer(t *testing.T) *httptest.Server {
	r := mux.NewRouter()
	srv := newServer(t)

	r.HandleFunc("/public/rest/v2/events", srv.returnCIS1Events).Methods(http.MethodGet)
	r.HandleFunc("/events/v1/events/central", srv.returnCIS2Events).Methods(http.MethodGet)

	return httptest.NewServer(r)
}

func (s *server) returnCIS1Events(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("type")
	if eventType != "SUBACCOUNT_DELETION" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var (
		page int
		size int
		err  error
	)

	pageSize := r.URL.Query().Get("resultsPerPage")
	if pageSize == "" {
		size = 10
	} else {
		size, err = strconv.Atoi(pageSize)
		require.NoError(s.t, err)
	}

	pageNum := r.URL.Query().Get("page")
	if pageNum == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(pageNum)
		require.NoError(s.t, err)
	}

	events := chunk(size, cis1Events())

	var response = fmt.Sprintf(`{
	  	"events":[%s],
	  	"totalResults":30,
	  	"totalPages":%d
	}`, strings.Join(events[page-1], ","), len(events))

	_, err = w.Write([]byte(response))
	if err != nil {
		s.t.Errorf("fakeCisServer (endpoint 1.0) cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) returnCIS2Events(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("eventType")
	if eventType != "Subaccount_Deletion" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var (
		page int
		size int
		err  error
	)

	pageSize := r.URL.Query().Get("pageSize")
	if pageSize == "" {
		size = 10
	} else {
		size, err = strconv.Atoi(pageSize)
		require.NoError(s.t, err)
	}

	pageNum := r.URL.Query().Get("pageNum")
	if pageNum == "" {
		page = 0
	} else {
		page, err = strconv.Atoi(pageNum)
		require.NoError(s.t, err)
	}

	events := chunk(size, cis2Events())

	// CIS 2.0 API counts pages from 0 (not from 1) - last page is always empty
	events = append(events, []string{})

	var response = fmt.Sprintf(`{
		"total": 30,
		"totalPages": %d,
		"pageNum": %d,
		"morePages": %t,
		"events": [%s]
	}`, len(events)-1, page, page < len(events)-1, strings.Join(events[page], ","))

	_, err = w.Write([]byte(response))
	if err != nil {
		s.t.Errorf("fakeCisServer (endpoint 2.0) cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func chunk(amount int, data []string) [][]string {
	var divided [][]string

	for i := 0; i < len(data); i += amount {
		end := i + amount
		if end > len(data) {
			end = len(data)
		}
		divided = append(divided, data[i:end])
	}

	return divided
}

func cis1Events() []string {
	var events []string
	instances := fixInstances()

	for index, event := range cis1EventsData {
		events = append(events, fmt.Sprintf(event, instances[index].SubAccountID))
	}

	return events
}

func cis2Events() []string {
	var events []string
	instances := fixInstances()

	for index, event := range cis2EventsData {
		events = append(events, fmt.Sprintf(event, instances[index].SubAccountID, instances[index].SubAccountID))
	}

	return events
}

var (
	cis1EventsData = []string{
		`{"id":1719072,"type":"SUBACCOUNT_DELETION","timestamp":"1597317000944","eventData":"{\"name\":\"khbf7phgg7\",\"globalAccountGuid\":\"91dd5c36-2bb5-44d1-b3a8-e02770facf51\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597316977503}"}`,
		`{"id":1719069,"type":"SUBACCOUNT_DELETION","timestamp":"1597314982043","eventData":"{\"name\":\"t5j61zb4p2\",\"globalAccountGuid\":\"91dd5c36-2bb5-44d1-b3a8-e02770facf51\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597314926602}"}`,
		`{"id":1719067,"type":"SUBACCOUNT_DELETION","timestamp":"1597314924791","eventData":"{\"name\":\"lin0imn4fb\",\"globalAccountGuid\":\"91dd5c36-2bb5-44d1-b3a8-e02770facf51\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597314881262}"}`,
		`{"id":1719061,"type":"SUBACCOUNT_DELETION","timestamp":"1597297466366","eventData":"{\"name\":\"ukjrziu08d\",\"globalAccountGuid\":\"91dd5c36-2bb5-44d1-b3a8-e02770facf51\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597297452389}"}`,
		`{"id":1719059,"type":"SUBACCOUNT_DELETION","timestamp":"1597297436914","eventData":"{\"name\":\"cod2x4h0ci\",\"globalAccountGuid\":\"91dd5c36-2bb5-44d1-b3a8-e02770facf51\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597297421600}"}`,
		`{"id":1719055,"type":"SUBACCOUNT_DELETION","timestamp":"1597293446478","eventData":"{\"name\":\"yd4c00ac2\",\"globalAccountGuid\":\"SecAutoTests\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597293442141}"}`,
		`{"id":1719053,"type":"SUBACCOUNT_DELETION","timestamp":"1597292280095","eventData":"{\"name\":\"y3ace6bee\",\"globalAccountGuid\":\"e1782da7-3b06-402a-b938-1580642218b9\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597292269877}"}`,
		`{"id":1719052,"type":"SUBACCOUNT_DELETION","timestamp":"1597292280095","eventData":"{\"name\":\"y4dc95b78\",\"globalAccountGuid\":\"e1782da7-3b06-402a-b938-1580642218b9\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597292273036}"}`,
		`{"id":1719048,"type":"SUBACCOUNT_DELETION","timestamp":"1597291279743","eventData":"{\"name\":\"y1f38b01f\",\"globalAccountGuid\":\"e1782da7-3b06-402a-b938-1580642218b9\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597291238630}"}`,
		`{"id":1719049,"type":"SUBACCOUNT_DELETION","timestamp":"1597291279743","eventData":"{\"name\":\"y683f8089\",\"globalAccountGuid\":\"e1782da7-3b06-402a-b938-1580642218b9\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597291241858}"}`,
		`{"id":1719045,"type":"SUBACCOUNT_DELETION","timestamp":"1597289789748","eventData":"{\"name\":\"y8f87ad8e\",\"globalAccountGuid\":\"gbaastest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597289729343}"}`,
		`{"id":1719043,"type":"SUBACCOUNT_DELETION","timestamp":"1597289539857","eventData":"{\"name\":\"yf8809d18\",\"globalAccountGuid\":\"gbaastest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597289501966}"}`,
		`{"id":1719041,"type":"SUBACCOUNT_DELETION","timestamp":"1597288377116","eventData":"{\"name\":\"y168efc34\",\"globalAccountGuid\":\"sap\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597288352324}"}`,
		`{"id":1719040,"type":"SUBACCOUNT_DELETION","timestamp":"1597288377116","eventData":"{\"name\":\"y6189cca2\",\"globalAccountGuid\":\"sap\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597288355796}"}`,
		`{"id":1719037,"type":"SUBACCOUNT_DELETION","timestamp":"1597284697757","eventData":"{\"name\":\"y88ea6997\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597284647642}"}`,
		`{"id":1719032,"type":"SUBACCOUNT_DELETION","timestamp":"1597282739858","eventData":"{\"name\":\"yffed5901\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597282692650}"}`,
		`{"id":1719030,"type":"SUBACCOUNT_DELETION","timestamp":"1597280624603","eventData":"{\"name\":\"y66e408bb\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597280564719}"}`,
		`{"id":1719028,"type":"SUBACCOUNT_DELETION","timestamp":"1597276747286","eventData":"{\"name\":\"y11e3382d\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276634965}"}`,
		`{"id":1719027,"type":"SUBACCOUNT_DELETION","timestamp":"1597276699184","eventData":"{\"name\":\"y11e3382d\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276634965}"}`,
		`{"id":1719025,"type":"SUBACCOUNT_DELETION","timestamp":"1597276581772","eventData":"{\"name\":\"y7124b1c8\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276478185}"}`,
		`{"id":1719024,"type":"SUBACCOUNT_DELETION","timestamp":"1597276225524","eventData":"{\"name\":\"y0623815e\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276204829}"}`,
		`{"id":1719023,"type":"SUBACCOUNT_DELETION","timestamp":"1597276225524","eventData":"{\"name\":\"y969c9ccf\",\"globalAccountGuid\":\"SAPconnectivity\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597276198808}"}`,
		`{"id":1719017,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597273899794\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275444310}"}`,
		`{"id":1719014,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274313852\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275450936}"}`,
		`{"id":1719022,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274304218\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275446042}"}`,
		`{"id":1719006,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274310628\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275447901}"}`,
		`{"id":1719000,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274038703\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275449573}"}`,
		`{"id":1719004,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274084715\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275447898}"}`,
		`{"id":1719003,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274081134\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275444311}"}`,
		`{"id":1719018,"type":"SUBACCOUNT_DELETION","timestamp":"1597275536377","eventData":"{\"name\":\"itest1597274297348\",\"globalAccountGuid\":\"cloudcockpittest\",\"tenantName\":\"%s\",\"origin\":null,\"timestamp\":1597275447899}"}`,
	}
	cis2EventsData = []string{
		`{
			"id": 639589,
			"actionTime": 1597322741353,
			"creationTime": 1597322742088,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "732aa8c5-d8dd-42a8-b0c4-90e6267b3016",
				"displayName": "TrialSub1",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "dom38484",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "732aa8c5-d8dd-42a8-b0c4-90e6267b3016",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 639522,
			"actionTime": 1597321854357,
			"creationTime": 1597321855065,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "6b4c5e5a-b4a1-4137-9d59-5cf62fbe2fab",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "89558c4btrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "6b4c5e5a-b4a1-4137-9d59-5cf62fbe2fab",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 638858,
			"actionTime": 1597311078287,
			"creationTime": 1597311079078,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "a45a997b-a426-41b5-853e-470d59412956",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "02553d35trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "a45a997b-a426-41b5-853e-470d59412956",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
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
		}`,
		`{
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
		}`,
		`{
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
		}`,
		`{
			"id": 628876,
			"actionTime": 1597073092177,
			"creationTime": 1597073093081,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "22b09b03-9e8b-4739-ab85-85d8ba97bd1d",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "39ea0bfetrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "22b09b03-9e8b-4739-ab85-85d8ba97bd1d",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 627716,
			"actionTime": 1597047718328,
			"creationTime": 1597047719085,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "96d19f17-a121-4419-80b1-dbec3233b5ca",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "b63f3421trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "96d19f17-a121-4419-80b1-dbec3233b5ca",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 625718,
			"actionTime": 1596960307478,
			"creationTime": 1596960308089,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "9d12a33c-5a9c-4322-b24c-d453e79c0ba3",
				"displayName": "Account-Dev",
				"subaccountDescription": "Development Subaccount",
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "398648d5-dff1-4308-8e81-3428b53c92f7",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "9d12a33c-5a9c-4322-b24c-d453e79c0ba3",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 621236,
			"actionTime": 1596792581434,
			"creationTime": 1596792582331,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "9d12a33c-5a9c-4322-b24c-d453e79c0ba3",
				"displayName": "Account-Dev",
				"subaccountDescription": "Development Subaccount",
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "09836aac-92cd-4a37-a28d-6b53521cc6ad",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "9d12a33c-5a9c-4322-b24c-d453e79c0ba3",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 617314,
			"actionTime": 1596695398708,
			"creationTime": 1596695399299,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "2d7ef7c1-9895-4d10-ba34-54246cd1f6f3",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "ef316e53trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "2d7ef7c1-9895-4d10-ba34-54246cd1f6f3",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 615041,
			"actionTime": 1596630357377,
			"creationTime": 1596630358080,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "9490f34d-6cd7-4d89-bf6b-e03c694a720c",
				"displayName": "Test_ME",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "TEST-ME-SPC-CLD",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "9490f34d-6cd7-4d89-bf6b-e03c694a720c",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 614985,
			"actionTime": 1596629729399,
			"creationTime": 1596629730104,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "ee2cdb33-3f98-4108-8c40-4e1870a6ecd5",
				"displayName": "i501632-fuck",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "i501632-fuck",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "ee2cdb33-3f98-4108-8c40-4e1870a6ecd5",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 614067,
			"actionTime": 1596615922755,
			"creationTime": 1596615923310,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "c0f22882-1ea0-4f8b-a850-a2b83226d307",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "048025d8trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "c0f22882-1ea0-4f8b-a850-a2b83226d307",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 614064,
			"actionTime": 1596615909394,
			"creationTime": 1596615910266,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "c0f22882-1ea0-4f8b-a850-a2b83226d307",
				"displayName": "nabeel-test",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "nabeeltestt",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "c0f22882-1ea0-4f8b-a850-a2b83226d307",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 609957,
			"actionTime": 1596522951476,
			"creationTime": 1596522952233,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "e63bd660-35f6-48d4-bf2a-8b77233b947e",
				"displayName": "z_sa",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "dsds",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "e63bd660-35f6-48d4-bf2a-8b77233b947e",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 605716,
			"actionTime": 1596367854558,
			"creationTime": 1596367855057,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "261ba8b6-6b33-47e7-9558-3bccb237a920",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "31f4a136trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "261ba8b6-6b33-47e7-9558-3bccb237a920",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 605715,
			"actionTime": 1596367853478,
			"creationTime": 1596367854068,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "261ba8b6-6b33-47e7-9558-3bccb237a920",
				"displayName": "trial-aws2",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "jklkjlkjl",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "261ba8b6-6b33-47e7-9558-3bccb237a920",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 605714,
			"actionTime": 1596367853363,
			"creationTime": 1596367854057,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "261ba8b6-6b33-47e7-9558-3bccb237a920",
				"displayName": "trial-aws3",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "trial-aws3-canary-vered-s",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "261ba8b6-6b33-47e7-9558-3bccb237a920",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 601724,
			"actionTime": 1596184712873,
			"creationTime": 1596184713273,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "6e0bbab7-8475-4616-8a62-969b9c5c4c40",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "718bd8bdtrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "6e0bbab7-8475-4616-8a62-969b9c5c4c40",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 595710,
			"actionTime": 1596027654527,
			"creationTime": 1596027655094,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "66f583c4-2752-446a-89d0-c6b4270db5cf",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "835876dbtrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "66f583c4-2752-446a-89d0-c6b4270db5cf",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 595707,
			"actionTime": 1596027641653,
			"creationTime": 1596027642279,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "66f583c4-2752-446a-89d0-c6b4270db5cf",
				"displayName": "subaccount2",
				"subaccountDescription": "string",
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "835876dbtrialsa2",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "66f583c4-2752-446a-89d0-c6b4270db5cf",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 594668,
			"actionTime": 1596009648611,
			"creationTime": 1596009649067,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "3dfea6f5-0183-49e6-b639-1da3c02d2916",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "fba37631trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "3dfea6f5-0183-49e6-b639-1da3c02d2916",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 592381,
			"actionTime": 1595946062732,
			"creationTime": 1595946063076,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "6d0de353-d5ef-4400-830b-5519588cb538",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "a82684abtrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "6d0de353-d5ef-4400-830b-5519588cb538",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 591018,
			"actionTime": 1595926842908,
			"creationTime": 1595926843309,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "673d8298-e639-4889-8f38-73c7a526f29e",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "578472fetrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "673d8298-e639-4889-8f38-73c7a526f29e",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 584181,
			"actionTime": 1595677554846,
			"creationTime": 1595677555088,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "d1583185-55f5-4791-be33-4b7583edd27b",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "88082afatrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "d1583185-55f5-4791-be33-4b7583edd27b",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 584180,
			"actionTime": 1595677554755,
			"creationTime": 1595677555077,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "9e680c4b-47e1-4498-b7d7-88f6ec2df1aa",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "b9c1c00dtrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "9e680c4b-47e1-4498-b7d7-88f6ec2df1aa",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 584179,
			"actionTime": 1595677554751,
			"creationTime": 1595677555067,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "a8c21ef2-138f-4b0e-9a3f-3925b72c96a2",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "0397e6cetrial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "a8c21ef2-138f-4b0e-9a3f-3925b72c96a2",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 584175,
			"actionTime": 1595677541448,
			"creationTime": 1595677542081,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "d1583185-55f5-4791-be33-4b7583edd27b",
				"displayName": "subaccount2",
				"subaccountDescription": "string",
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "88082afatrialsa2",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "d1583185-55f5-4791-be33-4b7583edd27b",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
		`{
			"id": 584120,
			"actionTime": 1595676654819,
			"creationTime": 1595676655062,
			"details": {
				"description": "Subaccount deleted.",
				"guid": "%s",
				"parentGuid": "1c9ddf1a-8ed9-4975-99f8-e7979fce14c4",
				"displayName": "trial",
				"subaccountDescription": null,
				"region": "eu10-canary",
				"jobLocation": null,
				"subdomain": "a85af412trial",
				"betaEnabled": false,
				"expiryDate": null
			},
			"globalAccountGUID": "1c9ddf1a-8ed9-4975-99f8-e7979fce14c4",
			"entityId": "%s",
			"entityType": "Subaccount",
			"eventOrigin": "accounts-service",
			"eventType": "Subaccount_Deletion"
		}`,
	}
)
