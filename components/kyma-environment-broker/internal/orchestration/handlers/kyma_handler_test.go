package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHandler_AttachRoutes(t *testing.T) {
	t.Run("upgrade", func(t *testing.T) {
		// given
		db := storage.NewMemoryStorage()
		logs := logrus.New()
		q := process.NewQueue(&testExecutor{}, logs)
		kymaHandler := handlers.NewKymaHandler(db.Orchestrations(), q, logs)

		params := orchestration.Parameters{
			Targets: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						RuntimeID: "test",
					},
				},
			},
		}
		p, err := json.Marshal(&params)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/upgrade/kyma", bytes.NewBuffer(p))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		kymaHandler.AttachRoutes(router)

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
func TestHandler_KymaVersion(t *testing.T) {
	t.Run("kyma version validation", func(t *testing.T) {
		// // given
		// db := storage.NewMemoryStorage()
		// logs := logrus.New()
		// q := process.NewQueue(&testExecutor{}, logs)
		// kymaHandler := handlers.NewKymaHandler(db.Orchestrations(), q, logs)

		//  // test semantic version
		//  // Exists: https://github.com/kyma-project/kyma/releases/tag/1.18.0
		//	require.NoError(t, kymaHandler.ValidateKymaVersion("1.18.0"))
		//
		//	err := kymaHandler.ValidateKymaVersion("0.12.34")
		//	require.NotNil(t, err)
		//	require.Contains(t, err.Error(), "not found")
		//
		//	// test PR- version
		//	// Exists: https://github.com/kyma-project/kyma/pull/10542
		//	require.NoError(t, kymaHandler.ValidateKymaVersion("PR-10542"))
		//
		//	err = kymaHandler.ValidateKymaVersion("PR-0")
		//	require.NotNil(t, err)
		//	require.Contains(t, err.Error(), "not found")
		//
		//	// test <branch name>-<commit hash> version
		//	// Exists: https://github.com/kyma-project/kyma/commit/f5e6d75
		//	require.NoError(t, kymaHandler.ValidateKymaVersion("master-f5e6d75"))
		//
		//	err = kymaHandler.ValidateKymaVersion("master-123456")
		//	require.NotNil(t, err)
		//	require.Contains(t, err.Error(), "not present on branch")
		//
		//	err = kymaHandler.ValidateKymaVersion("release-0.4-f5e6d75")
		//	require.NotNil(t, err)
		//	require.Contains(t, err.Error(), "not present on branch")
	})
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
