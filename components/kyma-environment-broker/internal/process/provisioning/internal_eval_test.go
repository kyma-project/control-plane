package provisioning

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/gorilla/mux"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type evaluationRepository map[int64]*avs.BasicEvaluationCreateResponse

type mockAvsService struct {
	server     *httptest.Server
	evals      evaluationRepository
	isInternal bool
	t          *testing.T
}

const dummyStrAvsTest = "dummy"

func TestInternalEvaluationStep_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	provisioningOperation := fixOperationCreateRuntime(t, broker.AzurePlanID, "westeurope")

	inputCreator := newInputCreator()
	provisioningOperation.InputCreator = inputCreator

	err := memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
	assert.NoError(t, err)

	mockOauthServer := newMockAvsOauthServer()
	defer mockOauthServer.Close()
	mockAvsSvc := newMockAvsService(t, true)
	mockAvsSvc.startServer()
	defer mockAvsSvc.server.Close()
	avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
	avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	assert.NoError(t, err)
	avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
	internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
	ies := NewInternalEvaluationStep(avsDel, internalEvalAssistant)

	// when
	logger := log.WithFields(logrus.Fields{"step": "TEST"})
	provisioningOperation, repeat, err := ies.Run(provisioningOperation, logger)

	//then
	assert.NoError(t, err)
	assert.Equal(t, 0*time.Second, repeat)

	inDB, err := memoryStorage.Operations().GetProvisioningOperationByID(provisioningOperation.ID)
	assert.NoError(t, err)
	assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AvsEvaluationInternalId)
}

func TestInternalEvaluationStep_WhenOperationIsRepeatedWithIdPresent(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	provisioningOperation := fixOperationCreateRuntime(t, broker.AzurePlanID, "westeurope")
	_, id := generateId()
	provisioningOperation.Avs.AvsEvaluationInternalId = id

	inputCreator := newInputCreator()
	provisioningOperation.InputCreator = inputCreator

	err := memoryStorage.Operations().InsertProvisioningOperation(provisioningOperation)
	assert.NoError(t, err)

	mockOauthServer := newMockAvsOauthServer()
	defer mockOauthServer.Close()
	mockAvsServer := newMockAvsService(t, true)
	mockAvsServer.startServer()
	defer mockAvsServer.server.Close()
	avsConfig := avsConfig(mockOauthServer, mockAvsServer.server)
	avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	assert.NoError(t, err)
	avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
	internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
	ies := NewInternalEvaluationStep(avsDel, internalEvalAssistant)

	// when
	logger := log.WithFields(logrus.Fields{"step": "TEST"})
	provisioningOperation, repeat, err := ies.Run(provisioningOperation, logger)

	//then
	assert.NoError(t, err)
	assert.Equal(t, 0*time.Second, repeat)

	inDB, err := memoryStorage.Operations().GetProvisioningOperationByID(provisioningOperation.ID)
	assert.NoError(t, err)
	assert.Equal(t, inDB.Avs.AvsEvaluationInternalId, id)
}

func newMockAvsOauthServer() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"access_token": "90d64460d14870c08c81352a05dedd3465940a7c", "scope": "user", "token_type": "bearer", "expires_in": 86400}`))
		}))
}

func newMockAvsService(t *testing.T, isInternal bool) *mockAvsService {
	return &mockAvsService{
		evals:      make(evaluationRepository, 0),
		isInternal: isInternal,
		t:          t,
	}
}

func (svc *mockAvsService) startServer() {
	r := mux.NewRouter()
	r.HandleFunc("/", svc.handleCreateEvaluation).Methods(http.MethodPost)
	r.HandleFunc("/{id}/tag", svc.handleAddTag).Methods(http.MethodPost)
	svc.server = httptest.NewServer(r)
}

func (svc *mockAvsService) handleCreateEvaluation(w http.ResponseWriter, r *http.Request) {
	assert.Equal(svc.t, r.Header.Get("Content-Type"), "application/json")
	dec := json.NewDecoder(r.Body)
	var requestObj avs.BasicEvaluationCreateRequest
	err := dec.Decode(&requestObj)
	assert.NoError(svc.t, err)

	if svc.isInternal {
		assert.Empty(svc.t, requestObj.URL)
	} else {
		assert.NotEmpty(svc.t, requestObj.URL)
	}

	evalCreateResponse := createResponseObj(requestObj, svc.t)
	svc.evals[evalCreateResponse.Id] = evalCreateResponse

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(evalCreateResponse); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *mockAvsService) handleAddTag(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var requestObj *avs.Tag
	err := dec.Decode(&requestObj)
	assert.NoError(svc.t, err)

	vars := mux.Vars(r)
	evalId, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	_, exists := svc.evals[evalId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}
	svc.evals[evalId].Tags = append(svc.evals[evalId].Tags, requestObj)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(svc.evals[evalId]); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func avsConfig(mockOauthServer *httptest.Server, mockAvsServer *httptest.Server) avs.Config {
	return avs.Config{
		OauthTokenEndpoint:     mockOauthServer.URL,
		OauthUsername:          dummyStrAvsTest,
		OauthPassword:          dummyStrAvsTest,
		OauthClientId:          dummyStrAvsTest,
		ApiEndpoint:            mockAvsServer.URL,
		DefinitionType:         avs.DefinitionType,
		InternalTesterAccessId: 1234,
		InternalTesterService:  "",
		InternalTesterTags:     []*avs.Tag{},
		ExternalTesterAccessId: 5678,
		ExternalTesterService:  dummyStrAvsTest,
		ExternalTesterTags: []*avs.Tag{
			{
				Content:      dummyStrAvsTest,
				TagClassId:   123,
				TagClassName: dummyStrAvsTest,
			},
		},
		GroupId:                     5555,
		ParentId:                    9101112,
		AdditionalTagsEnabled:       true,
		GardenerSeedNameTagClassId:  111111,
		GardenerShootNameTagClassId: 111112,
		RegionTagClassId:            111113,
	}
}

func createResponseObj(requestObj avs.BasicEvaluationCreateRequest, t *testing.T) *avs.BasicEvaluationCreateResponse {
	parseInt, err := strconv.ParseInt(requestObj.Threshold, 10, 64)
	assert.NoError(t, err)

	timeUnixEpoch, id := generateId()

	evalCreateResponse := &avs.BasicEvaluationCreateResponse{
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
		Threshold:                  parseInt,
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

func generateId() (int64, int64) {
	timeUnixEpoch := time.Now().Unix()
	id := time.Now().Unix()
	return timeUnixEpoch, id
}
