package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"go.uber.org/zap"

	skrsvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/svc"

	skrpvc "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/pvc"

	skrnode "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/skr/node"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/keb"

	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"
	"k8s.io/client-go/util/workqueue"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/mux"

	log "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/service"

	gardenersecret "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/secret"
	gardenershoot "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/gardener/shoot"
	kmcprocess "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/process"

	"github.com/kelseyhightower/envconfig"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/env"
	"github.com/kyma-project/control-plane/components/kyma-metrics-collector/options"
	gocache "github.com/patrickmn/go-cache"
)

const (
	metricsPath        = "/metrics"
	healthzPath        = "/healthz"
	edpCredentialsFile = "/edp-credentials/token"
)

func main() {

	opts := options.ParseArgs()
	logger := log.NewLogger(opts.LogLevel)
	logger.Infof("Starting application with options: %v", opts.String())

	cfg := new(env.Config)
	if err := envconfig.Process("", cfg); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load env config")
	}

	// Load public cloud specs
	publicCloudSpecs, err := kmcprocess.LoadPublicCloudSpecs(cfg)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load public cloud spec")
	}
	logger.Debugf("public cloud spec: %v", publicCloudSpecs)

	secretClient, err := gardenersecret.NewClient(opts)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Generate client for gardener secrets")
	}

	shootClient, err := gardenershoot.NewClient(opts)
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Generate client for gardener shoots")
	}

	// Create a client for KEB communication
	kebConfig := new(keb.Config)
	if err := envconfig.Process("", kebConfig); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load KEB config")
	}
	kebClient := keb.NewClient(kebConfig, logger)
	logger.Debugf("keb config: %v", kebConfig)

	// Creating cache with no expiration and the data will never be cleaned up
	cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

	// Creating EDP client
	edpConfig := new(edp.Config)
	if err := envconfig.Process("", edpConfig); err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load EDP config")
	}

	// read the token from the mounted secret
	token, err := getEDPToken()
	if err != nil {
		logger.With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Fatal("Load EDP token")
	}
	edpConfig.Token = token

	edpClient := edp.NewClient(edpConfig, logger)

	queue := workqueue.NewDelayingQueue()

	kmcProcess := kmcprocess.Process{
		KEBClient:       kebClient,
		ShootClient:     shootClient,
		SecretClient:    secretClient,
		EDPClient:       edpClient,
		Logger:          logger,
		Providers:       publicCloudSpecs,
		Cache:           cache,
		ScrapeInterval:  opts.ScrapeInterval,
		Queue:           queue,
		WorkersPoolSize: opts.WorkerPoolSize,
		NodeConfig:      skrnode.Config{},
		PVCConfig:       skrpvc.Config{},
		SvcConfig:       skrsvc.Config{},
	}

	// Start execution
	go kmcProcess.Start()

	// add debug service.
	if opts.DebugPort > 0 {
		enableDebugging(opts.DebugPort, logger)
	}
	router := mux.NewRouter()
	router.Path(healthzPath).HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.Path(metricsPath).Handler(promhttp.Handler())

	kmcSvr := service.Server{
		Addr:   fmt.Sprintf(":%d", opts.ListenAddr),
		Logger: logger,
		Router: router,
	}

	// Start a server to cater to the metrics and healthz endpoints
	kmcSvr.Start()
}

func enableDebugging(debugPort int, log *zap.SugaredLogger) {
	debugRouter := mux.NewRouter()
	// for security reason we always listen on localhost
	debugSvc := service.Server{
		Addr:   fmt.Sprintf("127.0.0.1:%d", debugPort),
		Logger: log,
		Router: debugRouter,
	}

	debugRouter.HandleFunc("/debug/pprof/", pprof.Index)
	debugRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	debugRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
	debugRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	debugRouter.HandleFunc("/debug/pprof/trace", pprof.Trace)
	debugRouter.Handle("/debug/pprof/block", pprof.Handler("block"))
	debugRouter.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	debugRouter.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	debugRouter.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	go func() {
		debugSvc.Start()
	}()
}

// getEDPToken read the EDP token from the mounted secret file
func getEDPToken() (string, error) {
	token, err := os.ReadFile(edpCredentialsFile)
	if err != nil {
		return "", err
	}
	trimmedToken := strings.TrimSuffix(string(token), "\n")
	return trimmedToken, nil
}
