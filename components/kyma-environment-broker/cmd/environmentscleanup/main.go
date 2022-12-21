package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"
	"k8s.io/client-go/dynamic"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/environmentscleanup"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

type config struct {
	MaxAgeHours   time.Duration `envconfig:"default=24h"`
	LabelSelector string        `envconfig:"default=owner.do-not-delete!=true"`
	Gardener      gardener.Config
	Database      storage.Config
	Broker        broker.ClientConfig
	Provisioner   provisionerConfig
}

type provisionerConfig struct {
	URL          string `envconfig:"default=kcp-provisioner:3000"`
	QueryDumping bool   `envconfig:"default=false"`
}

func main() {

	time.Sleep(20 * time.Second)

	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	if err != nil {
		fatalOnError(fmt.Errorf("while loading cleanup config: %w", err))
	}

	clusterCfg, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	if err != nil {
		fatalOnError(fmt.Errorf("while creating Gardener cluster config: %w", err))
	}
	cli, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		fatalOnError(fmt.Errorf("while creating Gardener client: %w", err))
	}
	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.Gardener.Project)
	shootClient := cli.Resource(gardener.ShootResource).Namespace(gardenerNamespace)

	ctx := context.Background()
	brokerClient := broker.NewClient(ctx, cfg.Broker)
	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioner.URL, cfg.Provisioner.QueryDumping)

	// create storage
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	db, conn, err := storage.NewFromConfig(cfg.Database, events.Config{}, cipher, log.WithField("service", "storage"))
	if err != nil {
		fatalOnError(err)
	}
	dbStatsCollector := sqlstats.NewStatsCollector("broker", conn)
	prometheus.MustRegister(dbStatsCollector)

	logger := log.New()

	svc := environmentscleanup.NewService(shootClient, brokerClient, provisionerClient, db.Instances(), logger, cfg.MaxAgeHours, cfg.LabelSelector)
	err = svc.PerformCleanup()
	if err != nil {
		fatalOnError(err)
	}

	err = conn.Close()
	if err != nil {
		fatalOnError(err)
	}

	// do not use defer, close must be done before halting
	err = cleaner.Halt()
	if err != nil {
		fatalOnError(err)
	}

	log.Info("Kyma Environments cleanup performed successfully")

	time.Sleep(5 * time.Second)
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
