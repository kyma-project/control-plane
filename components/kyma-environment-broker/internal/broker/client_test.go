package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

const (
	fixInstanceID = "72b83910-ac12-4dcb-b91d-960cca2b36abx"
	fixRuntimeID  = "24da44ea-0295-4b1c-b5c1-6fd26efa4f24"
	fixOpID       = "04f91bff-9e17-45cb-a246-84d511274ef1"

	gcpPlanID   = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	azurePlanID = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
)

func TestClient_Deprovision(t *testing.T) {
	t.Run("should return deprovisioning operation ID on success", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(nil)
		defer testServer.Close()

		config := NewClientConfig(testServer.URL)
		client := NewClientWithPoller(context.Background(), *config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: azurePlanID,
		}

		// when
		opID, err := client.Deprovision(instance)

		// then
		assert.NoError(t, err)
		assert.Equal(t, fixOpID, opID)
	})

	t.Run("should return error on failed request execution", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(requestFailureServerError)
		defer testServer.Close()

		config := NewClientConfig(testServer.URL)

		client := NewClientWithPoller(context.Background(), *config, NewPassthroughPoller())

		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: gcpPlanID,
		}

		// when
		opID, err := client.Deprovision(instance)

		// then
		assert.Error(t, err)
		assert.Len(t, opID, 0)
	})
}

func TestClient_ExpirationRequest(t *testing.T) {

	t.Run("should return true on successfully commenced suspension", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: TrialPlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.NoError(t, err)
		assert.True(t, suspensionUnderWay)
	})

	t.Run("should return error when trying to make other plan than trial expired", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: azurePlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.Error(t, err)
		assert.False(t, suspensionUnderWay)
	})

	t.Run("should return error when update fails", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(requestFailureServerError)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())

		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: TrialPlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.Error(t, err)
		assert.False(t, suspensionUnderWay)
	})

	t.Run("should return false on unprocessable entity", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(requestFailureUnprocessableEntity)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: TrialPlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.Error(t, err)
		assert.False(t, suspensionUnderWay)
	})

	t.Run("should return true for non-existent instanceId and false for existing", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		// when
		response, err := client.GetInstanceRequest("non-existent")

		// then
		assert.NoError(t, err)
		assert.Equal(t, response.StatusCode, http.StatusNotFound)

		// when
		responseOtherThanNotFound, err := client.GetInstanceRequest("real")

		// then
		assert.NoError(t, err)
		assert.NotEqual(t, responseOtherThanNotFound.StatusCode, http.StatusNotFound)
	})
}

func fixHTTPServer(requestFailureFunc func(http.ResponseWriter, *http.Request)) *httptest.Server {
	if requestFailureFunc != nil {
		r := mux.NewRouter()
		r.HandleFunc("/oauth/v2/service_instances/{instance_id}", requestFailureFunc).Methods(http.MethodDelete)
		r.HandleFunc("/oauth/v2/service_instances/{instance_id}", requestFailureFunc).Methods(http.MethodPatch)
		return httptest.NewServer(r)
	}

	r := mux.NewRouter()
	r.HandleFunc("/oauth/v2/service_instances/{instance_id}", deprovision).Methods(http.MethodDelete)
	r.HandleFunc("/oauth/v2/service_instances/{instance_id}", serviceUpdateWithExpiration).Methods(http.MethodPatch)
	r.HandleFunc("/oauth/v2/service_instances/{instance_id}", getInstance).Methods(http.MethodGet)
	return httptest.NewServer(r)
}

func serviceUpdateWithExpiration(w http.ResponseWriter, r *http.Request) {
	responseDTO := serviceUpdatePatchDTO{}
	err := json.NewDecoder(r.Body).Decode(&responseDTO)

	validRequest := err == nil && responseDTO.PlanID == TrialPlanID &&
		*responseDTO.Parameters.Expired && !*responseDTO.Context.Active

	if !validRequest {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"operation": "%s"}`, fixOpID)))
}

func deprovision(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	_, okServiceID := params["service_id"]
	if !okServiceID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, okPlanID := params["plan_id"]
	if !okPlanID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf(`{"operation": "%s"}`, fixOpID)))
}

func requestFailureServerError(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func requestFailureUnprocessableEntity(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnprocessableEntity)
}

func getInstance(w http.ResponseWriter, r *http.Request) {
	instance := path.Base(r.URL.Path)
	if instance == "non-existent" {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.Write([]byte(fmt.Sprintf(`{"instanceID": "%s"}`, instance)))
		w.WriteHeader(http.StatusOK)
	}
}
