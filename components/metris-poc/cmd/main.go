package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/metris-poc/pkg/env"

	"github.com/gorilla/mux"
	system_info "github.com/kyma-project/control-plane/components/metris-poc/pkg/system-info"
	log "github.com/sirupsen/logrus"
)

const (
	livenessURI  = "/healthz"
	readinessURI = "/readyz"
)

type options struct {
	requestTimeout int
}

func main() {
	fmt.Println("Starting POC")
	requestTimeout := flag.Int("requestTimeout", 1, "Timeout for services.")
	flag.Parse()

	cfg := env.GetConfig()
	opts := &options{
		requestTimeout: *requestTimeout,
	}

	// Create client for gardener
	gardenerClient, err := createClientForGardener()

	// Create client for KEB

	// Create client for SKRs

	server := &http.Server{
		Addr:         ":8080",
		Handler:      NewHandler(),
		WriteTimeout: time.Duration(opts.requestTimeout) * time.Second,
	}

	go start(server)
}

func createClientForGardener()

func start(server *http.Server) {
	if server == nil {
		log.Error("cannot start a nil HTTP server")
		return
	}

	if err := server.ListenAndServe(); err != nil {
		log.Errorf("failed to start server: %v", err)
	}
}

func NewHandler() http.Handler {
	router := mux.NewRouter()

	router.Path("/systemInfo").Handler(NewSystemStatsHandler()).Methods(http.MethodGet)

	router.Path(livenessURI).Handler(CheckHealth()).Methods(http.MethodGet)

	router.Path(readinessURI).Handler(CheckHealth()).Methods(http.MethodGet)

	return router
}

func NewSystemStatsHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		sysInfo, err := system_info.GetSystemInfo()
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("failed to get sys info: %v", err)))
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		sysInfoBytes, err := json.Marshal(sysInfo)
		if err != nil {
			log.Errorf("failed to Marshal the response: %v", err)
			return
		}
		_, err = writer.Write([]byte(sysInfoBytes))
		if err != nil {
			log.Errorf("failed to write in the response: %v", err)
		}
	})
}

func CheckHealth() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		return
	})
}
