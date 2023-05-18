package main

import (
	"context"
	"log"

	btpmanager "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/btpmanager/credentials"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Config struct {
	Database                             storage.Config
	Events                               events.Config
	Provisioner                          input.Config
	DryRun                               bool   `envconfig:"default=true"`
	BtpManagerSecretWatcherAddr          string `envconfig:"default=0"`
	BtpManagerSecretWatcherComponentName string `envconfig:"default=NA"`
	WatcherEnabled                       bool   `envconfig:"default=false"`
	JobEnabled                           bool   `envconfig:"default=false"`
	JobInterval                          int    `envconfig:"default=24"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	logs.Info("runtime-reconciler started")
	logs.Info("runtime-reconciler debug version: 1")

	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "RUNTIME_RECONCILER")
	fatalOnError(err)
	logs.Info("runtime-reconciler config loaded")
	if !cfg.JobEnabled && !cfg.WatcherEnabled {
		logs.Info("both job and listener are disabled, module stopped.")
		return
	}
	logs.Infof("runtime-listener runing as dry run? %t", cfg.DryRun)

	cipher := storage.NewEncrypter(cfg.Database.SecretKey)

	db, _, err := storage.NewFromConfig(cfg.Database, cfg.Events, cipher, logs.WithField("service", "storage"))
	fatalOnError(err)
	logs.Info("runtime-reconciler connected to database")

	kcpK8sConfig, err := config.GetConfig()
	fatalOnError(err)
	kcpK8sClient, err := client.New(kcpK8sConfig, client.Options{})
	fatalOnError(err)

	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioner.URL, false)

	btpOperatorManager := btpmanager.NewManager(ctx, kcpK8sClient, db.Instances(), logs, cfg.DryRun, provisionerClient)

	logs.Infof("job enabled? %t", cfg.JobEnabled)
	if cfg.JobEnabled {
		btpManagerCredentialsJob := btpmanager.NewJob(btpOperatorManager, logs)
		logs.Infof("runtime-reconciler created job every %d m", cfg.JobInterval)
		btpManagerCredentialsJob.Start(cfg.JobInterval)
	}

	logs.Infof("watcher enabled? %t", cfg.WatcherEnabled)
	if cfg.WatcherEnabled {
		btpManagerCredentialsWatcher := btpmanager.NewWatcher(ctx, cfg.BtpManagerSecretWatcherAddr, cfg.BtpManagerSecretWatcherComponentName, btpOperatorManager, logs)
		logs.Infof("runtime-reconciler created watcher %s on %s", cfg.BtpManagerSecretWatcherComponentName, cfg.BtpManagerSecretWatcherAddr)
		go btpManagerCredentialsWatcher.ReactOnSkrEvent()
	}

	<-ctx.Done()
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
