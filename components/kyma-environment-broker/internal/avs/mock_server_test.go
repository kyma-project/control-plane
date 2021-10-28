package avs

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

const (
	accessToken = "1234abcd"
	tokenType   = "test"
)

// MockAvsEvaluationRepository represents BasicEvaluations in AVS
// where BasicEvals is mapping BasicEvaluation ID to BasicEvaluation (Subevaluation) definition
// and ParentIDrefs is mapping CompoundEvaluation ID (parentID) to BasicEvaluations (Subevaluations) IDs
type MockAvsEvaluationRepository struct {
	BasicEvals   map[int64]*BasicEvaluationCreateResponse
	EvalSet      map[int64]bool
	ParentIDrefs map[int64][]int64
}

type MockAvsServer struct {
	T            *testing.T
	Evaluations  *MockAvsEvaluationRepository
	TokenExpired int
}

func NewMockAvsServer(t *testing.T) *MockAvsServer {
	return &MockAvsServer{
		T: t,
		Evaluations: &MockAvsEvaluationRepository{
			BasicEvals:   make(map[int64]*BasicEvaluationCreateResponse, 0),
			EvalSet:      make(map[int64]bool, 0),
			ParentIDrefs: make(map[int64][]int64, 0),
		},
	}
}

func FixMockAvsServer(srv *MockAvsServer) *httptest.Server {
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

func (s *MockAvsServer) token(w http.ResponseWriter, _ *http.Request) {
	token := oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: "",
		Expiry:       time.Time{},
	}

	response, err := json.Marshal(token)
	assert.NoError(s.T, err)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(response)
	assert.NoError(s.T, err)
}

func (s *MockAvsServer) hasAccess(token string) bool {
	if s.TokenExpired > 0 {
		s.TokenExpired--
		return false
	}
	if token == fmt.Sprintf("%s %s", tokenType, accessToken) {
		return true
	}

	return false
}

func (er *MockAvsEvaluationRepository) addEvaluation(parentID int64, eval *BasicEvaluationCreateResponse) {
	er.BasicEvals[eval.Id] = eval
	er.EvalSet[eval.Id] = true
	er.ParentIDrefs[parentID] = append(er.ParentIDrefs[parentID], eval.Id)
}

func (er *MockAvsEvaluationRepository) removeParentRef(parentID, evalID int64) {
	refs := er.ParentIDrefs[parentID]

	for i, evalWithRef := range refs {
		if evalID == evalWithRef {
			refs[i] = refs[len(refs)-1]
			er.ParentIDrefs[parentID] = refs[:len(refs)-1]
		}
	}
}

func (s *MockAvsServer) createEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.T, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj BasicEvaluationCreateRequest
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.T, err)

	evalCreateResponse := s.createResponseObj(requestObj)
	s.Evaluations.addEvaluation(requestObj.ParentId, evalCreateResponse)

	createdEval := s.Evaluations.BasicEvals[evalCreateResponse.Id]
	responseObjAsBytes, _ := json.Marshal(createdEval)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)

	w.WriteHeader(http.StatusOK)
}

func (s *MockAvsServer) getEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.T, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	response, exists := s.Evaluations.BasicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

func (s *MockAvsServer) addTagToEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.T, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj *Tag
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.T, err)

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	evaluation, exists := s.Evaluations.BasicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	evaluation.Tags = append(evaluation.Tags, requestObj)

	responseObjAsBytes, _ := json.Marshal(evaluation)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

func (s *MockAvsServer) setStatus(w http.ResponseWriter, r *http.Request) {
	assert.Equal(s.T, r.Header.Get("Content-Type"), "application/json")
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var requestObj string
	err := json.NewDecoder(r.Body).Decode(&requestObj)
	assert.NoError(s.T, err)

	if !ValidStatus(requestObj) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["evalId"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	evaluation, exists := s.Evaluations.BasicEvals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	evaluation.Status = requestObj

	responseObjAsBytes, _ := json.Marshal(evaluation)
	_, err = w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

func (s *MockAvsServer) deleteEvaluation(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.T, err)

	if _, exists := s.Evaluations.BasicEvals[id]; exists {
		delete(s.Evaluations.BasicEvals, id)
		delete(s.Evaluations.EvalSet, id)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *MockAvsServer) removeReferenceFromParentEval(w http.ResponseWriter, r *http.Request) {
	if !s.hasAccess(r.Header.Get("Authorization")) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	parentID, err := strconv.ParseInt(vars["parentId"], 10, 64)
	assert.NoError(s.T, err)

	evalID, err := strconv.ParseInt(vars["evalId"], 10, 64)
	assert.NoError(s.T, err)

	_, exists := s.Evaluations.ParentIDrefs[parentID]
	if !exists {
		resp := avsNonSuccessResp{
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf("Evaluation %d does not contain subevaluation %d", parentID, evalID),
		}
		bytes, _ := json.Marshal(resp)
		_, err := w.Write(bytes)
		assert.NoError(s.T, err)
		w.WriteHeader(http.StatusBadRequest)
	}

	s.Evaluations.removeParentRef(parentID, evalID)
}

func FixTag() *Tag {
	return &Tag{
		Content:    "test-tag",
		TagClassId: 111111,
	}
}

func (s *MockAvsServer) createResponseObj(requestObj BasicEvaluationCreateRequest) *BasicEvaluationCreateResponse {
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

func (s *MockAvsServer) generateId() (int64, int64) {
	for {
		timeUnixEpoch := time.Now().Unix()
		id := rand.Int63() + time.Now().Unix()
		if _, exists := s.Evaluations.EvalSet[id]; !exists {
			return timeUnixEpoch, id
		}
	}
}
