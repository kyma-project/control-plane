package main

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cis"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	ClientVersion string
	CIS           cis.Config
	Database      storage.Config
	Broker        broker.ClientConfig
}

func main() {
	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	// create logs
	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	// create CIS client
	var client cis.CisClient
	switch cfg.ClientVersion {
	case "v1.0":
		client = cis.NewClientVer1(ctx, cfg.CIS, logs)
	case "v2.0":
		client = cis.NewClient(ctx, cfg.CIS, logs)
	default:
		logs.Fatalf("Client version %s is not supported", cfg.ClientVersion)
	}

	// create storage connection
	db, _, err := storage.NewFromConfig(cfg.Database, logs.WithField("service", "storage"))
	fatalOnError(err)

	// create broker client
	brokerClient := broker.NewClient(ctx, cfg.Broker)

	// create SubAccountCleanerService and execute process
	sacs := cis.NewSubAccountCleanupService(client, brokerClient, db.Instances(), logs)
	fatalOnError(sacs.Run())
}

func fatalOnError(err error) {
	if err != nil {
		logrus.Fatal(err)
	}
}
