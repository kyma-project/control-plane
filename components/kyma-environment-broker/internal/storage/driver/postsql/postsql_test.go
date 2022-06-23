package postsql_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() { os.Exit(exitVal) }()

	ctx := context.Background()

	cleanupNetwork, err := storage.SetupTestNetworkForDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanupNetwork()

	containerCleanupFunc, cfg, err := storage.InitTestDBContainer(log.Printf, ctx, "test_DB_1")
	if err != nil {
		log.Fatal(err)
	}
	defer containerCleanupFunc()

	_, err = storage.SetupTestDBTables(cfg.ConnectionURL())
	if err != nil {
		log.Fatal(err)
	}

	exitVal = m.Run()
}
