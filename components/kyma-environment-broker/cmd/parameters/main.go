package main

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/migrations"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type config struct {
	Database storage.Config
}

// parameters migration fetches all operations from storage and rewrites provisioning parameters nested in JSON data field to a separate field
func main() {
	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(errors.Wrap(err, "while loading config"))
	log := logrus.New()

	db, _, err := storage.NewFromConfig(cfg.Database, log.WithField("service", "storage"))
	fatalOnError(err)

	err = migrations.NewParametersMigration(db.Operations(), log).Migrate()
	fatalOnError(err)
}

func fatalOnError(err error) {
	if err != nil {
		logrus.Fatal(err)
	}
}
