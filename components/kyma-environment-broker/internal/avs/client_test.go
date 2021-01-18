package avs

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	parentEvaluationID     = 42
	evaluationName         = "test_evaluation"
	existingEvaluationName = "test-eval-name"
	accessToken            = "1234abcd"
	tokenType              = "test"
)

func TestClient_CreateEvaluation(t *testing.T) {
	t.Run("create evaluation the first time", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
		}, logrus.New())
		assert.NoError(t, err)

		// When
		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     evaluationName,
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, evaluationName, response.Name)
		assert.NotEmpty(t, server.evaluations.parentIDrefs[parentEvaluationID])
	})

	t.Run("create evaluation with token reset", func(t *testing.T) {
		// Given
		server := newServer(t)
		server.tokenExpired = 1
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
		}, logrus.New())
		assert.NoError(t, err)

		// When
		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     evaluationName,
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "test_evaluation", response.Name)
		assert.NotEmpty(t, server.evaluations.parentIDrefs[parentEvaluationID])
	})

	t.Run("401 error during creating evaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		server.tokenExpired = 2
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		_, err = client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name: "test_evaluation",
		})

		// Then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func TestClient_DeleteEvaluation(t *testing.T) {
	t.Run("should delete existing evaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name: "test_evaluation",
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.evaluations.basicEvals)
	})

	t.Run("should return error when trying to delete not existing evaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		_, err = client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(123)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.evaluations.basicEvals[123])
	})
}

func TestClient_GetEvaluation(t *testing.T) {
	t.Run("should get existing evaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:        "test_evaluation_create",
			Description: "custom description",
		})
		assert.NoError(t, err)

		// When
		getResp, err := client.GetEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, getResp, resp)
	})

	t.Run("should return error when trying to get not existing evaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		_, err = client.GetEvaluation(1)

		// Then
		assert.Contains(t, err.Error(), "404")
	})
}

func TestClient_Status(t *testing.T) {
	t.Run("should get status", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		getResp, err := client.GetEvaluation(resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, getResp.Status, StatusActive)
	})

	t.Run("should update status", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		resp, err = client.SetStatus(resp.Id, StatusDeleted)

		// Then
		assert.NoError(t, err)
		assert.Equal(t, resp.Status, StatusDeleted)
	})

	t.Run("should not update invalid status", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{})
		assert.NoError(t, err)

		// When
		resp, err = client.SetStatus(resp.Id, "")

		// Then
		assert.Contains(t, err.Error(), "500")
	})
}

func TestClient_RemoveReferenceFromParentEval(t *testing.T) {
	t.Run("should remove reference from parent eval", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		resp, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(parentEvaluationID, resp.Id)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.evaluations.parentIDrefs[parentEvaluationID])
	})
	t.Run("should return error when wrong api url provided", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", "http://not-existing"),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(parentEvaluationID, 111)

		// then
		assert.Error(t, err)
	})
	t.Run("should return error when parent evaluation does not contain subevaluation", func(t *testing.T) {
		// Given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		// When
		err = client.RemoveReferenceFromParentEval(int64(9999), 111)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

}

func TestClient_AddTag(t *testing.T) {
	t.Run("should add tag to existing evaluation", func(t *testing.T) {
		// given
		server := newServer(t)
		mockServer := fixHTTPServer(server)
		client, err := NewClient(context.TODO(), Config{
			OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
			ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
			ParentId:           parentEvaluationID,
		}, logrus.New())
		assert.NoError(t, err)

		response, err := client.CreateEvaluation(&BasicEvaluationCreateRequest{
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})
		assert.NoError(t, err)

		fixedTag := fixTag()

		// when
		eval, err := client.AddTag(response.Id, fixedTag)

		// then
		assert.NoError(t, err)
		assert.Equal(t, fixedTag, eval.Tags[0])
	})
}

// evaluationRepository represents BasicEvaluations in AVS
//where basicEvals is mapping BasicEvaluation ID to BasicEvaluation (Subevaluation) definition
//and parentIDrefs is mapping CompoundEvaluation ID (parentID) to BasicEvaluations (Subevaluations) IDs
type evaluationRepository struct {
	basicEvals   map[int64]*BasicEvaluationCreateResponse
	evalSet      map[int64]bool
	parentIDrefs map[int64][]int64
}

func (er *evaluationRepository) addEvaluation(parentID int64, eval *BasicEvaluationCreateResponse) {
	er.basicEvals[eval.Id] = eval
	er.evalSet[eval.Id] = true
	er.parentIDrefs[parentID] = append(er.parentIDrefs[parentID], eval.Id)
}

func (er *evaluationRepository) removeParentRef(parentID, evalID int64) {
	refs := er.parentIDrefs[parentID]

	for i, evalWithRef := range refs {
		if evalID == evalWithRef {
			refs[i] = refs[len(refs)-1]
			er.parentIDrefs[parentID] = refs[:len(refs)-1]
		}
	}
}

type server struct {
	t            *testing.T
	evaluations  *evaluationRepository
	tokenExpired int
}

func newServer(t *testing.T) *server {
	return &server{
		t: t,
		evaluations: &evaluationRepository{
			basicEvals:   make(map[int64]*BasicEvaluationCreateResponse, 0),
			evalSet:      make(map[int64]bool, 0),
			parentIDrefs: make(map[int64][]int64, 0),
		},
	}
}

func fixHTTPServer(srv *server) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/oauth/token", srv.token).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata", srv.createEvaluation).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}", srv.deleteEvaluation).Methods(http.MethodDelete)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}", srv.getEvaluation).Methods(http.MethodGet)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}/tag", srv.addTagToEvaluation).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}/lifecycle", srv.setStatus).Methods(http.MethodPut)
	r.HandleFunc("/api/v2/evaluationmetadata/{parentId}/child/{evalId}", srv.removeReferenceFromParentEval).Methods(http.MethodDelete)

	return httptest.NewServer(r)
}

func (s *server) token(w http.ResponseWriter, _ *http.Request) {
	token := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	response, err := json.Marshal(token)
	assert.NoError(s.t, err)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(response)
	assert.NoError(s.t, err)

	w.WriteHeader(http.StatusOK)
}

func (s *server) hasAccess(token string) bool {
	if s.tokenExpired > 0 {
		s.tokenExpired--
		return false
	}
	if token == fmt.Sprintf("%s %s", tokenType, accessToken) {
		return true
	}

	return false
}

func (s *server) createEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj BasicEvaluationCreateRequest
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.t, err)

	evalCreateResponse := s.createResponseObj(requestObj)
	s.evaluations.addEvaluation(requestObj.ParentId, evalCreateResponse)

	createdEval := s.evaluations.basicEvals[evalCreateResponse.Id]
	responseObjAsBytes, _ := json.Marshal(createdEval)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)

	w.WriteHeader(http.StatusOK)
}

func (s *server) getEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	response, exists := s.evaluations.basicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)
}

func (s *server) addTagToEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj *Tag
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.t, err)

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	evaluation, exists := s.evaluations.basicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	evaluation.Tags = append(evaluation.Tags, requestObj)

	responseObjAsBytes, _ := json.Marshal(evaluation)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)
}

func (s *server) setStatus(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.t, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj string
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.t, err)

	if !ValidStatus(requestObj) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	evaluation, exists := s.evaluations.basicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	evaluation.Status = requestObj

	responseObjAsBytes, _ := json.Marshal(evaluation)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)
}

func (s *server) deleteEvaluation(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	if _, exists := s.evaluations.basicEvals[id]; exists {
		delete(s.evaluations.basicEvals, id)
		delete(s.evaluations.evalSet, id)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *server) removeReferenceFromParentEval(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	parentID, err := strconv.ParseInt(vars["parentId"], 10, 64)
	assert.NoError(s.t, err)

	evalID, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	_, exists := s.evaluations.parentIDrefs[parentID]
	if !exists {
		w.WriteHeader(http.StatusBadRequest)
	}

	s.evaluations.removeParentRef(parentID, evalID)
}

func fixTag() *Tag {
	return &Tag{
		Content:    "test-tag",
		TagClassId: 111111,
	}
}

func (s *server) createResponseObj(requestObj BasicEvaluationCreateRequest) *BasicEvaluationCreateResponse {
	parsedThreshold, err := strconv.ParseInt(requestObj.Threshold, 10, 64)
	if err != nil {
		parsedThreshold = int64(1234)
	}

	timeUnixEpoch, id := s.generateId()

	evalCreateResponse := &BasicEvaluationCreateResponse{
		DefinitionType:             requestObj.DefinitionType,
		Name:                       requestObj.Name,
		Description:                requestObj.Description,
		Service:                    requestObj.Service,
		URL:                        requestObj.URL,
		CheckType:                  requestObj.CheckType,
		Interval:                   requestObj.Interval,
		TesterAccessId:             requestObj.TesterAccessId,
		Timeout:                    requestObj.Timeout,
		ReadOnly:                   requestObj.ReadOnly,
		ContentCheck:               requestObj.ContentCheck,
		ContentCheckType:           requestObj.ContentCheck,
		Threshold:                  parsedThreshold,
		GroupId:                    requestObj.GroupId,
		Visibility:                 requestObj.Visibility,
		DateCreated:                timeUnixEpoch,
		DateChanged:                timeUnixEpoch,
		Owner:                      "abc@xyz.corp",
		Status:                     "ACTIVE",
		Alerts:                     nil,
		Tags:                       requestObj.Tags,
		Id:                         id,
		LegacyCheckId:              id,
		InternalInterval:           60,
		AuthType:                   "AUTH_NONE",
		IndividualOutageEventsOnly: false,
		IdOnTester:                 "",
	}
	return evalCreateResponse
}

func (s *server) generateId() (int64, int64) {
	for {
		timeUnixEpoch := time.Now().Unix()
		id := rand.Int63() + time.Now().Unix()
		if _, exists := s.evaluations.evalSet[id]; !exists {
			return timeUnixEpoch, id
		}
	}
}
