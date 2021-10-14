package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestKymaHandler_AttachRoutes(t *testing.T) {
	t.Run("upgrade", func(t *testing.T) {
		// given
		kHandler := fixKymaHandler(t)

		params := orchestration.Parameters{
			Targets: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						RuntimeID: "test",
					},
				},
			},
			Kubernetes: &orchestration.KubernetesParameters{
				KubernetesVersion: "",
			},
			Kyma: &orchestration.KymaParameters{
				Version: "",
			},
		}
		p, err := json.Marshal(&params)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/upgrade/kyma", bytes.NewBuffer(p))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kHandler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusAccepted, rr.Code)

		var out orchestration.UpgradeResponse

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.NotEmpty(t, out.OrchestrationID)
	})
}

// Testing Kyma Version is disabled due to GitHub API RATE limits
func TestKymaHandler_KymaVersion(t *testing.T) {
	t.Run("kyma version validation", func(t *testing.T) {
		// given
		kHandler := fixKymaHandler(t)

		// test semantic version
		// Exists: 1.18.0
		require.NoError(t, kHandler.ValidateKymaVersion("1.18.0"))

		err := kHandler.ValidateKymaVersion("0.12.34")
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "not found")

		// test PR- version
		// Exists: 10542
		require.NoError(t, kHandler.ValidateKymaVersion("PR-10542"))

		err = kHandler.ValidateKymaVersion("PR-0")
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "not found")

		// test <branch name>-<commit hash> version
		// Exists: main-f5e6d75
		require.NoError(t, kHandler.ValidateKymaVersion("main-f5e6d75"))

		err = kHandler.ValidateKymaVersion("main-123456")
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "not present on branch")

		err = kHandler.ValidateKymaVersion("release-0.4-f5e6d75")
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "not present on branch")
	})
}

func fixKymaHandler(t *testing.T) *kymaHandler {
	db := storage.NewMemoryStorage()
	logs := logrus.New()
	q := process.NewQueue(&testExecutor{}, logs)
	kHandler := NewKymaHandler(db.Orchestrations(), q, logs)

	// fix github client
	mockServer := fixGithubServer(t)
	baseUrl, _ := url.Parse(fmt.Sprintf("%s/", mockServer.URL))
	kHandler.gitClient.BaseURL = baseUrl

	return kHandler
}

func fixGithubServer(t *testing.T) *httptest.Server {
	r := mux.NewRouter()

	// initialize Git data
	mock := mockServer{
		T: t,
		tags: map[string]bool{
			"1.18.0": true,
		},
		pulls: map[int64]bool{
			10542: true,
		},
		branchCommit: map[string]map[string]bool{
			"main": map[string]bool{
				"f5e6d75": true,
			},
		},
	}

	// set routes
	r.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/tags/{tag}",
		internal.GitKymaProject, internal.GitKymaRepo), mock.getTags).Methods(http.MethodGet)
	r.HandleFunc(fmt.Sprintf("/repos/%s/%s/pulls/{pullId}",
		internal.GitKymaProject, internal.GitKymaRepo), mock.getPulls).Methods(http.MethodGet)
	r.HandleFunc(fmt.Sprintf("/repos/%v/%v/compare/{base}...{commit}",
		internal.GitKymaProject, internal.GitKymaRepo), mock.getDiff).Methods(http.MethodGet)

	return httptest.NewServer(r)
}

type mockServer struct {
	T            *testing.T
	tags         map[string]bool
	pulls        map[int64]bool
	branchCommit map[string]map[string]bool
}

func (s *mockServer) getTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	tag := vars["tag"]
	response := github.RepositoryRelease{}

	// check if tag exists
	_, exists := s.tags[tag]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	} else {
		response = github.RepositoryRelease{TagName: &tag}
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err := w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

func (s *mockServer) getPulls(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	pullId, _ := strconv.ParseInt(vars["pullId"], 10, 64)
	response := github.PullRequest{}

	// check if pull exists
	_, exists := s.pulls[pullId]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	} else {
		response = github.PullRequest{ID: &pullId}
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err := w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

func (s *mockServer) getDiff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	response := github.CommitsComparison{}

	// check if branch exists
	branch, commit := vars["base"], vars["commit"]
	branchCommits, exists := s.branchCommit[branch]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
	}

	// check if commit exists
	_, exists = branchCommits[commit]
	if !exists {
		response.Commits = []github.RepositoryCommit{
			{SHA: &commit},
		}
	}

	responseObjAsBytes, _ := json.Marshal(response)
	_, err := w.Write(responseObjAsBytes)
	assert.NoError(s.T, err)
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
