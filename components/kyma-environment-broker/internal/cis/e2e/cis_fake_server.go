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
	if eventType != "MASTER_SUBACCOUNT_DELETION" {
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
		`{"id":145385,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597913650896","eventData":"{\"globalAccountGuid\":\"9f023be7-4678-4e24-9a39-7755ca8b6891\",\"subaccountGuid\":\"%s\",\"platformID\":\"t45a53814\",\"subdomain\":\"c20a4d11-f452-4917-a6a7-3810e5f40487\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145383,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597913624896","eventData":"{\"globalAccountGuid\":\"cb76a519-79b4-4d66-bbd9-0bc4a82ced26\",\"subaccountGuid\":\"%s\",\"platformID\":\"8d7e4b47-6b67-49a7-b1d6-6ffe0423ee30\",\"subdomain\":\"dev-tenant-2\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145382,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597912900232","eventData":"{\"globalAccountGuid\":\"e065f1b9-94fc-4494-a009-ceb1cd7bc779\",\"subaccountGuid\":\"%s\",\"platformID\":\"2e97e6d7-7980-4f6c-84ed-31afef059907\",\"subdomain\":\"OQ-toni\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145381,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597912674809","eventData":"{\"globalAccountGuid\":\"almts\",\"subaccountGuid\":\"%s\",\"platformID\":\"a5392d50-1fca-460b-8d2f-7aea45950ae8\",\"subdomain\":\"orphantest\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145380,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597912194772","eventData":"{\"globalAccountGuid\":\"27c2ccb8-8916-a1e6-18af-99007dee6639\",\"subaccountGuid\":\"%s\",\"platformID\":\"b0c4bf65-804e-4c5d-935d-b54ef7a3c95e\",\"subdomain\":\"CustomUI\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145379,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597911775006","eventData":"{\"globalAccountGuid\":\"d4037436-f01a-4adc-aa2b-52836f459bfe\",\"subaccountGuid\":\"%s\",\"platformID\":\"1cb359a3-2d83-44f7-8ddb-2195fcba2f94\",\"subdomain\":\"release1-15rc1-test\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145378,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597911149713","eventData":"{\"globalAccountGuid\":\"1f599a9e-ab04-4948-809e-e8341f8e8eec\",\"subaccountGuid\":\"%s\",\"platformID\":\"bl9r8oey00\",\"subdomain\":\"057792f5-6b7a-4f58-a9ae-858025be9dd7\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145375,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597909968182","eventData":"{\"globalAccountGuid\":\"almts\",\"subaccountGuid\":\"%s\",\"platformID\":\"905b276d-5fbd-42c2-9dff-32ed80482bd0\",\"subdomain\":\"unsubscribed\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145374,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597909558868","eventData":"{\"globalAccountGuid\":\"b13eaa86-b117-406e-ac1d-0930b471d732\",\"subaccountGuid\":\"%s\",\"platformID\":\"bdfdd276-b8f4-4a18-af10-8bcebd4bd6b6\",\"subdomain\":\"sec-si-dec\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145371,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597909023236","eventData":"{\"globalAccountGuid\":\"6b04f2ee-f31a-450d-bf56-c802c7120f03\",\"subaccountGuid\":\"%s\",\"platformID\":\"9253015b-22f3-4fb3-a59f-58b2d85d0954\",\"subdomain\":\"testprovsetup\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145369,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597908235870","eventData":"{\"globalAccountGuid\":\"9f023be7-4678-4e24-9a39-7755ca8b6891\",\"subaccountGuid\":\"%s\",\"platformID\":\"t32a20882\",\"subdomain\":\"6974742f-17bf-4a9a-9985-5d8b6f2630ac\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145368,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597907366152","eventData":"{\"globalAccountGuid\":\"bpm\",\"subaccountGuid\":\"%s\",\"platformID\":\"tbbe19c7f\",\"subdomain\":\"d3a95b20-86d2-4d75-9a77-4637bb3397b7\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145367,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597907126299","eventData":"{\"globalAccountGuid\":\"bpm\",\"subaccountGuid\":\"%s\",\"platformID\":\"tcce6ace9\",\"subdomain\":\"0ddd2512-8ef0-4a0d-a0c2-4f4b2ec62213\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145366,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906863271","eventData":"{\"globalAccountGuid\":\"22cb7ffd-6f53-4b94-9915-b8a6dc6038f3\",\"subaccountGuid\":\"%s\",\"platformID\":\"8e8b4231-0224-441e-a017-234cdccd816e\",\"subdomain\":\"procurepartobedeleted\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145365,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906758247","eventData":"{\"globalAccountGuid\":\"22cb7ffd-6f53-4b94-9915-b8a6dc6038f3\",\"subaccountGuid\":\"%s\",\"platformID\":\"064adad0-6d0c-4afb-8439-6095d86d4dfd\",\"subdomain\":\"procurepardevc4\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145364,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906403195","eventData":"{\"globalAccountGuid\":\"423e5e64-1b67-474b-87ae-8c84070cdcaa\",\"subaccountGuid\":\"%s\",\"platformID\":\"4ae3bbb3-dc96-47b5-81f9-66b5fa030b11\",\"subdomain\":\"iotae-hotfixtmtest01\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145363,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906383102","eventData":"{\"globalAccountGuid\":\"423e5e64-1b67-474b-87ae-8c84070cdcaa\",\"subaccountGuid\":\"%s\",\"platformID\":\"2838513a-2585-4e98-b322-fac75345cf44\",\"subdomain\":\"iotae-hotfixtmtest02\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145362,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597906343187","eventData":"{\"globalAccountGuid\":\"423e5e64-1b67-474b-87ae-8c84070cdcaa\",\"subaccountGuid\":\"%s\",\"platformID\":\"e415c820-7895-41da-91cd-8f82a2b6f469\",\"subdomain\":\"iotae-hotfixtmtest03\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145361,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597905288293","eventData":"{\"globalAccountGuid\":\"9cae2ea3-f827-4e81-ad87-d354c9e376a8\",\"subaccountGuid\":\"%s\",\"platformID\":\"704548cc-5fdc-47e5-b03d-15445fb102f4\",\"subdomain\":\"irpachallenge--demo\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145360,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597902683098","eventData":"{\"globalAccountGuid\":\"cloudfnddm\",\"subaccountGuid\":\"%s\",\"platformID\":\"3ec68e7e-4c19-4c08-a417-680d7885d6ae\",\"subdomain\":\"AtestUnsubs\",\"region\":\"eu10-canary\"}"}`,
		`{"id":145359,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597898786106","eventData":"{\"globalAccountGuid\":\"bpm\",\"subaccountGuid\":\"%s\",\"platformID\":\"t5c59b178\",\"subdomain\":\"8d0f80c4-5780-4678-a2c3-4bc80dd46331\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145358,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597898546100","eventData":"{\"globalAccountGuid\":\"bpm\",\"subaccountGuid\":\"%s\",\"platformID\":\"t2b5e81ee\",\"subdomain\":\"83cfd327-b0f4-4036-87cf-48c58e0990ef\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145357,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597885135909","eventData":"{\"globalAccountGuid\":\"d2721a72-d9e1-40cb-ae73-53152254d8c1\",\"subaccountGuid\":\"%s\",\"platformID\":\"tc550e0c2\",\"subdomain\":\"a52f5dd6-314a-49a8-876f-10d0abefde1f\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145356,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597885106144","eventData":"{\"globalAccountGuid\":\"d2721a72-d9e1-40cb-ae73-53152254d8c1\",\"subaccountGuid\":\"%s\",\"platformID\":\"tb257d054\",\"subdomain\":\"01e7a8ef-4fd9-42d0-806b-122d727d7935\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145355,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597885075934","eventData":"{\"globalAccountGuid\":\"0d72937a-4797-41ea-8c9c-95ed2e4f9823\",\"subaccountGuid\":\"%s\",\"platformID\":\"t2c3345f7\",\"subdomain\":\"e4535720-55c8-4cd4-8ca4-af4f19facc11\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145354,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597884986187","eventData":"{\"globalAccountGuid\":\"0d72937a-4797-41ea-8c9c-95ed2e4f9823\",\"subaccountGuid\":\"%s\",\"platformID\":\"t5b347561\",\"subdomain\":\"df952150-386b-4e37-ac2c-011fd51953fb\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145353,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597884866410","eventData":"{\"globalAccountGuid\":\"0d72937a-4797-41ea-8c9c-95ed2e4f9823\",\"subaccountGuid\":\"%s\",\"platformID\":\"tc23d24db\",\"subdomain\":\"7b86bd36-8170-4adf-9948-827edd22a532\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145352,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597884776402","eventData":"{\"globalAccountGuid\":\"0d72937a-4797-41ea-8c9c-95ed2e4f9823\",\"subaccountGuid\":\"%s\",\"platformID\":\"tb53a144d\",\"subdomain\":\"c3e48642-28a7-43f8-9b64-1bfd8dbde12a\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145351,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597884686329","eventData":"{\"globalAccountGuid\":\"d2721a72-d9e1-40cb-ae73-53152254d8c1\",\"subaccountGuid\":\"%s\",\"platformID\":\"td5fd9da8\",\"subdomain\":\"54a1e542-964e-4f28-a366-0310e525f598\",\"region\":\"eu2-canary\"}"}`,
		`{"id":145350,"type":"MASTER_SUBACCOUNT_DELETION","timestamp":"1597884596099","eventData":"{\"globalAccountGuid\":\"d2721a72-d9e1-40cb-ae73-53152254d8c1\",\"subaccountGuid\":\"%s\",\"platformID\":\"ta2faad3e\",\"subdomain\":\"87b1d6f1-6a41-46bb-bf19-dc655ae241a8\",\"region\":\"eu2-canary\"}"}`,
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
