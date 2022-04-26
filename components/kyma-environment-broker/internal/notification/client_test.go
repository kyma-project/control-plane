package notification

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestClient_CreateEvent(t *testing.T) {
	// given
	server := fixHTTPServer(t)
	defer server.Close()

	client := NewClient(server.Client(), ClientConfig{URL: server.URL})

	// when
	tenants := []NotificationTenant{
		{
			InstanceID: "WEAJKG-INSTANCE-1",
			StartDate:  "2022-01-01T20:00:02Z",
		},
	}
	eventRequest := CreateEventRequest{
		OrchestrationID: "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP",
		EventType:       KymaMaintenanceNumber,
		Tenants:         tenants,
	}
	err := client.CreateEvent(eventRequest)

	// then
	assert.NoError(t, err)

	response, err := server.Client().Get(fmt.Sprintf("%s/get", server.URL))
	assert.NoError(t, err)

	var conf CreateEventRequest
	err = json.NewDecoder(response.Body).Decode(&conf)
	assert.NoError(t, err)

	assert.Equal(t, "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP", conf.OrchestrationID)
	assert.Equal(t, "1", conf.EventType)
	assert.Equal(t, "WEAJKG-INSTANCE-1", conf.Tenants[0].InstanceID)
	assert.Equal(t, "2022-01-01T20:00:02Z", conf.Tenants[0].StartDate)
}

func TestClient_UpdateEvent(t *testing.T) {
	// given
	server := fixHTTPServer(t)
	defer server.Close()

	client := NewClient(server.Client(), ClientConfig{URL: server.URL})

	// when
	tenants := []NotificationTenant{
		{
			InstanceID: "WEAJKG-INSTANCE-1",
			StartDate:  "2022-01-01T20:00:02Z",
			State:      UnderMaintenanceEventState,
		},
	}
	eventRequest := UpdateEventRequest{
		OrchestrationID: "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP",
		Tenants:         tenants,
	}
	err := client.UpdateEvent(eventRequest)

	// then
	assert.NoError(t, err)

	response, err := server.Client().Get(fmt.Sprintf("%s/get", server.URL))
	assert.NoError(t, err)

	var conf UpdateEventRequest
	err = json.NewDecoder(response.Body).Decode(&conf)
	assert.NoError(t, err)

	assert.Equal(t, "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP", conf.OrchestrationID)
	assert.Equal(t, "1", conf.Tenants[0].State)
	assert.Equal(t, "2022-01-01T20:00:02Z", conf.Tenants[0].StartDate)
}

func TestClient_CancelEvent(t *testing.T) {
	// given
	server := fixHTTPServer(t)
	defer server.Close()

	client := NewClient(server.Client(), ClientConfig{URL: server.URL})

	// when
	eventRequest := CancelEventRequest{
		OrchestrationID: "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP",
	}
	err := client.CancelEvent(eventRequest)

	// then
	assert.NoError(t, err)

	response, err := server.Client().Get(fmt.Sprintf("%s/get", server.URL))
	assert.NoError(t, err)

	var conf CancelEventRequest
	err = json.NewDecoder(response.Body).Decode(&conf)
	assert.NoError(t, err)

	assert.Equal(t, "ASKHGK-SAKJHTJ-ALKJSHT-HUZIUOP", conf.OrchestrationID)
}

type server struct {
	t       *testing.T
	request []byte
}

func fixHTTPServer(t *testing.T) *httptest.Server {
	s := server{t: t}

	r := mux.NewRouter()
	r.HandleFunc("/createMaintenanceEvent", s.authorized(s.createEvent)).Methods(http.MethodPost)
	r.HandleFunc("/updateMaintenanceEvent", s.authorized(s.updateEvent)).Methods(http.MethodPatch)
	r.HandleFunc("/cancelMaintenanceEvent", s.authorized(s.cancelEvent)).Methods(http.MethodPatch)

	r.HandleFunc("/get", s.getConfiguration).Methods(http.MethodGet)

	return httptest.NewServer(r)
}

func (s *server) authorized(pass func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//assume that comunication between application already built up
		pass(w, r)
	}
}

func (s *server) createEvent(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.t.Errorf("test server cannot read request body: %s", err)
		return
	}
	s.request = body
	w.WriteHeader(http.StatusOK)
}

func (s *server) updateEvent(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.t.Errorf("test server cannot read request body: %s", err)
		return
	}
	s.request = body
	w.WriteHeader(http.StatusOK)
}

func (s *server) cancelEvent(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.t.Errorf("test server cannot read request body: %s", err)
		return
	}
	s.request = body
	w.WriteHeader(http.StatusOK)
}

func (s *server) getConfiguration(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write(s.request)
	if err != nil {
		s.t.Errorf("test server cannot write response body: %s", err)
	}
}
