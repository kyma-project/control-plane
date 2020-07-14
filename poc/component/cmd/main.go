package main

import (
	"github.com/kyma-project/control-plane/poc/component/internal/handler"
	"github.com/kyma-project/control-plane/poc/component/internal/store"
	"github.com/pkg/errors"
	"github.com/vrischmann/envconfig"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type config struct {
	Address string `envconfig:"default=127.0.0.1:3001"`

	ConfigurationFilePath string `envconfig:"default=hack/config.yaml"`
}

func main() {
	cfg := config{}

	err := envconfig.InitWithPrefix(&cfg, "APP")
	exitOnError(err, "while loading app config")


	rtmStore := store.New(cfg.ConfigurationFilePath)
	rtmHandler := handler.New(rtmStore)
	router := mux.NewRouter()

	router.HandleFunc("/runtimes", rtmHandler.List).Methods(http.MethodGet)
	router.HandleFunc("/runtimes/{runtimeID}", rtmHandler.Get).Methods(http.MethodGet)

	log.Printf("API listening on %s", cfg.Address)
	err = http.ListenAndServe(cfg.Address, router)
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}
