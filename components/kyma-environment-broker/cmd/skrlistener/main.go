package main

import (
	"context"
	"fmt"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/events"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"log"
)

type Config struct {
	Database                              storage.Config
	Events                                events.Config
	BtpManagerSecretListenerAddr          string
	BtpManagerSecretListenerComponentName string
	DbInMemory                            bool
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	//For dev
	cfg.DbInMemory = true

	var db storage.BrokerStorage
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	if cfg.DbInMemory {
		db = storage.NewMemoryStorage()
	} else {
		store, _, err := storage.NewFromConfig(cfg.Database, cfg.Events, cipher, logs.WithField("service", "storage"))
		fatalOnError(err)
		db = store
	}

	fmt.Println(db.Instances())
	ctx.Done()
	//btpManagerSecretListener := skrlisteners.NewBtpManagerSecretListener(ctx, db.Instances(), cfg.BtpManagerSecretListenerAddr, cfg.BtpManagerSecretListenerComponentName, skrlisteners.NoVerify, logs)
	//go btpManagerSecretListener.ReactOnSkrEvent()
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
