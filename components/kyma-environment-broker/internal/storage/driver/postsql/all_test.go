package postsql_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/require"
)

const (
	initTests = "Initialization tests"
	instanceTests = "Instances tests"
	operationTests = "Operations tests"
	conflictTests = "Conflicts tests"
	orchestrationTests = "Orchestrations tests"
	runtimestateTests = "RuntimeStates tests"
	lmstenantTests = "LMS Tenants"
)

var testsRanInSuite bool

func TestPostgres(t *testing.T) {
	ctx := context.Background()

	cleanupNetwork, err := storage.EnsureTestNetworkForDB(t, ctx)
	require.NoError(t, err)
	defer cleanupNetwork()

	containerCleanupFunc, cfg, err := storage.InitTestDBContainer(t, ctx, "test_DB_1")
	require.NoError(t, err)
	defer containerCleanupFunc()

	_, err = storage.InitTestDBTables(t, cfg.ConnectionURL())
	require.NoError(t, err)

	for n, v := range fixTests() {
		t.Run(n, v)
	}

	testsRanInSuite = true
}

func fixTests() map[string]func(t *testing.T) {
	return map[string]func(t *testing.T){
		initTests: TestInitialization,
		instanceTests: TestInstance,
		operationTests: TestOperation,
		conflictTests: TestConflict,
		orchestrationTests: TestOrchestration,
		runtimestateTests: TestRuntimeState,
		lmstenantTests: TestLMSTenant,
	}
}