package avs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	evaluationID       = 997
	parentEvaluationID = 42
	accessToken        = "1234abcd"
	tokenType          = "test"
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
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "test_evaluation", response.Name)
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
			Name:     "test_evaluation",
			ParentId: parentEvaluationID,
		})

		// Then
		assert.NoError(t, err)
		assert.Equal(t, "test_evaluation", response.Name)
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
	t.Run("delete existing evaluation", func(t *testing.T) {
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
			Name: "test_evaluation",
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(evaluationID)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.evaluation)
	})

	t.Run("delete not existing evaluation", func(t *testing.T) {
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
			Name: "test_evaluation",
		})
		assert.NoError(t, err)

		// When
		err = client.DeleteEvaluation(123)

		// Then
		assert.NoError(t, err)
		assert.Empty(t, server.evaluation[123])
	})
}

func TestClient_RemoveReferenceFromParentEval(t *testing.T) {
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
		Name: "test_evaluation",
	})
	assert.NoError(t, err)

	// When
	err = client.RemoveReferenceFromParentEval(evaluationID)

	// Then
	assert.NoError(t, err)
	assert.Empty(t, server.evaluation[evaluationID])
}

func TestClient_RemoveReferenceFromParentEval_WrongApiURLError(t *testing.T) {
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
	err = client.RemoveReferenceFromParentEval(evaluationID)
	assert.Error(t, err)
}

type server struct {
	t            *testing.T
	evaluation   map[int64]int64
	tokenExpired int
}

func newServer(t *testing.T) *server {
	return &server{
		t:          t,
		evaluation: make(map[int64]int64, 0),
	}
}

func fixHTTPServer(srv *server) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/oauth/token", srv.token).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata", srv.createEvaluation).Methods(http.MethodPost)
	r.HandleFunc("/api/v2/evaluationmetadata/{evalId}", srv.deleteEvaluation).Methods(http.MethodDelete)
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

	s.evaluation[evaluationID] = parentEvaluationID

	response := BasicEvaluationCreateResponse{
		Name: requestObj.Name,
	}
	responseObjAsBytes, _ := json.Marshal(response)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.t, err)

	w.WriteHeader(http.StatusOK)
}

func (s *server) deleteEvaluation(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	if _, ok := s.evaluation[id]; ok {
		s.evaluation = map[int64]int64{}
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
	n, err := strconv.ParseInt(vars["parentId"], 10, 64)
	assert.NoError(s.t, err)

	id, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.t, err)

	if s.evaluation[id] == n {
		delete(s.evaluation, id)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
