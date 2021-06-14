package main

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/job"
	"github.com/kyma-project/control-plane/components/provisioners-model-migrating-job/internal/persistence/dbconnection"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

const (
	databaseConnectionRetries        = 20
	connStringFormat          string = "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s"

	maxErrors = 10
)

type config struct {
	Database struct {
		User     string `envconfig:"default=postgres"`
		Password string `envconfig:"default=password"`
		Host     string `envconfig:"default=localhost"`
		Port     string `envconfig:"default=5432"`
		Name     string `envconfig:"default=provisioner"`
		SSLMode  string `envconfig:"default=disable"`
	}
}

func main() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	exitOnError(err, "Failed to load application config")

	connString := fmt.Sprintf(connStringFormat, cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode)

	connection, err := dbconnection.InitializeDatabaseConnection(connString, databaseConnectionRetries)

	exitOnError(err, "Failed to connect to database")

	factory := dbconnection.NewFactory(connection)

	migrator := job.NewProviderConfigMigrator(factory, maxErrors)

	log.Info("Starting provider config migration job")
	err = migrator.Do()

	exitOnError(err, "Migration job failed")

	log.Info("Finished migrating job successfully")
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}
