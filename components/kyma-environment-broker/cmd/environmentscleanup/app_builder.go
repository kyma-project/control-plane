package main

import (
	"context"
	"fmt"
	"time"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/gocraft/dbr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/environmentscleanup"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/schema-migrator/cleaner"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/client-go/dynamic"
)

type config struct {
	MaxAgeHours   time.Duration `envconfig:"default=24h"`
	LabelSelector string        `envconfig:"default=owner.do-not-delete"`
	Gardener      gardener.Config
	Database      storage.Config
	Broker        broker.ClientConfig
	Provisioner   provisionerConfig
}

type AppBuilder struct {
	cfg                 config
	shootClient         dynamic.ResourceInterface
	brokerRuntimeClient environmentscleanup.BrokerRuntimesClient
	db                  storage.BrokerStorage
	conn                *dbr.Connection
	logger              *logrus.Logger

	brokerClient      *broker.Client
	provisionerClient provisioner.Client
}

type Job interface {
	Run() error
}

func NewAppBuilder() AppBuilder {
	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	if err != nil {
		FatalOnError(fmt.Errorf("while loading cleanup config: %w", err))
	}

	return AppBuilder{}
}

func (b *AppBuilder) withGardenerClient() {
	clusterCfg, err := gardener.NewGardenerClusterConfig(b.cfg.Gardener.KubeconfigPath)
	if err != nil {
		FatalOnError(fmt.Errorf("while creating Gardener cluster config: %w", err))
	}
	cli, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		FatalOnError(fmt.Errorf("while creating Gardener client: %w", err))
	}
	gardenerNamespace := fmt.Sprintf("garden-%s", b.cfg.Gardener.Project)
	b.shootClient = cli.Resource(gardener.ShootResource).Namespace(gardenerNamespace)
}

func (b *AppBuilder) withBrokerClient() {
	ctx := context.Background()
	b.brokerClient = broker.NewClient(ctx, b.cfg.Broker)
	b.provisionerClient = provisioner.NewProvisionerClient(b.cfg.Provisioner.URL, b.cfg.Provisioner.QueryDumping)

	clientCfg := clientcredentials.Config{
		ClientID:     b.cfg.Broker.ClientID,
		ClientSecret: b.cfg.Broker.ClientSecret,
		TokenURL:     b.cfg.Broker.TokenURL,
		Scopes:       []string{b.cfg.Broker.Scope},
	}
	httpClientOAuth := clientCfg.Client(ctx)
	httpClientOAuth.Timeout = 30 * time.Second
	b.brokerRuntimeClient = runtime.NewClient(b.cfg.Broker.URL, httpClientOAuth)
}

func (b *AppBuilder) withStorage() {
	// Init Storage
	cipher := storage.NewEncrypter(b.cfg.Database.SecretKey)
	var err error
	b.db, b.conn, err = storage.NewFromConfig(b.cfg.Database, events.Config{}, cipher, log.WithField("service", "storage"))
	if err != nil {
		FatalOnError(err)
	}
	dbStatsCollector := sqlstats.NewStatsCollector("broker", b.conn)
	prometheus.MustRegister(dbStatsCollector)

	b.logger = log.New()
}

func (b *AppBuilder) Cleanup() {
	err := b.conn.Close()
	if err != nil {
		FatalOnError(err)
	}

	// do not use defer, close must be done before halting
	err = cleaner.Halt()
	if err != nil {
		FatalOnError(err)
	}
}

func (b *AppBuilder) Create() Job {
	return environmentscleanup.NewService(
		b.shootClient,
		b.brokerClient,
		b.brokerRuntimeClient,
		b.provisionerClient,
		b.db.Instances(),
		b.logger,
		b.cfg.MaxAgeHours,
		b.cfg.LabelSelector,
	)
}
