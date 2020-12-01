package servicemanager

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	username = "some-user"
	password = "some-passwd"
)

func Test_ListOfferings(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/service_offerings", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	so, err := client.ListOfferings()

	// then
	require.NoError(t, err)
	require.NotNil(t, so)
}

func Test_ListOfferingsByName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/service_offerings", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "name eq 'xsuaa'", r.URL.Query().Get("fieldQuery"))

		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	so, err := client.ListOfferingsByName("xsuaa")

	// then
	require.NoError(t, err)
	require.NotNil(t, so)
}

func Test_ListPlans(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//then
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/service_plans", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "name eq 'application' and service_offering_id eq 'off-id'", r.URL.Query().Get("fieldQuery"))

		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	so, err := client.ListPlansByName("application", "off-id")

	// then
	require.NoError(t, err)
	require.NotNil(t, so)
}

func Test_Provision(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// then
		defer r.Body.Close()
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/osb/broker1234/v2/service_instances/instance-id-001", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		reqData := extractBody(t, r)

		assert.Equal(t, "s-id", reqData["service_id"])
		assert.Equal(t, "p-id", reqData["plan_id"])

		assert.Equal(t, "true", r.URL.Query().Get("accepts_incomplete"))

		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	so, err := client.Provision("broker1234", ProvisioningInput{
		ProvisionRequest: ProvisionRequest{
			ServiceID: "s-id",
			PlanID:    "p-id",
		},
		ID: "instance-id-001",
	}, true)

	// then
	require.NoError(t, err)
	require.NotNil(t, so)
}

func Test_Deprovision(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		// then
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/osb/broker1234/v2/service_instances/instance-id-001", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "s-id", r.URL.Query().Get("service_id"))
		assert.Equal(t, "p-id", r.URL.Query().Get("plan_id"))

		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	so, err := client.Deprovision(InstanceKey{
		BrokerID:   "broker1234",
		InstanceID: "instance-id-001",
		ServiceID:  "s-id",
		PlanID:     "p-id",
	}, false)

	// then
	require.NoError(t, err)
	require.NotNil(t, so)
}

func Test_LastInstanceOperation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		// then
		assertBasicAuth(t, r)
		assert.Equal(t, "/v1/osb/broker1234/v2/service_instances/instance-id-001/last_operation", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "s-id", r.URL.Query().Get("service_id"))
		assert.Equal(t, "p-id", r.URL.Query().Get("plan_id"))
		assert.Equal(t, "op-id", r.URL.Query().Get("operation"))

		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	client := New(Credentials{
		Username: username,
		Password: password,
		URL:      ts.URL,
	})

	// when
	resp, err := client.LastInstanceOperation(InstanceKey{
		BrokerID:   "broker1234",
		InstanceID: "instance-id-001",
		ServiceID:  "s-id",
		PlanID:     "p-id",
	}, "op-id")

	// then
	require.NoError(t, err)
}

func extractBody(t *testing.T, r *http.Request) map[string]interface{} {
	reqData := map[string]interface{}{}
	bytes, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	err = json.Unmarshal(bytes, &reqData)
	require.NoError(t, err)

	return reqData
}

func assertBasicAuth(t *testing.T, r *http.Request) {
	auth := r.Header.Get("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	assert.Equal(t, expectedAuth, auth)
}
