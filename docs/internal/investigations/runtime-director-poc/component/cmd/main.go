package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kyma-incubator/compass/components/director/pkg/executor"
	"github.com/kyma-incubator/compass/components/director/pkg/signal"
	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-director-poc/component/internal/handler"
	"github.com/kyma-project/control-plane/docs/internal/investigations/runtime-director-poc/component/internal/store"
	"github.com/pkg/errors"
	"github.com/vrischmann/envconfig"
)

type config struct {
	Address string `envconfig:"default=127.0.0.1:3001"`

	ConfigurationFilePath   string        `envconfig:"default=hack/config.yaml"`
	ConfigurationFileReload time.Duration `envconfig:"default=10s"`
}

func main() {
	cfg := config{}

	err := envconfig.InitWithPrefix(&cfg, "APP")
	exitOnError(err, "while loading app config")

	stopCh := signal.SetupChannel()

	rtmStore := store.New(cfg.ConfigurationFilePath)
	rtmHandler := handler.New(rtmStore)
	router := mux.NewRouter()

	executor.NewPeriodic(cfg.ConfigurationFileReload, func(stopCh <-chan struct{}) {
		if err := rtmStore.LoadConfig(); err != nil {
			exitOnError(err, "Error from Configuration reloader")
		}
		log.Println("Successfully reloaded configuration file")
	}).Run(stopCh)

	router.HandleFunc("/runtimes", rtmHandler.List).Methods(http.MethodGet)
	router.HandleFunc("/runtimes/{runtimeID}", rtmHandler.Get).Methods(http.MethodGet)

	srv := &http.Server{Addr: cfg.Address, Handler: router}

	go func() {
		<-stopCh
		// Interrupt signal received - shut down the servers
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v\n", err)
		}
	}()

	log.Printf("Running server on %s...\n", cfg.Address)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP server ListenAndServe: %v\n", err)
	}
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}
