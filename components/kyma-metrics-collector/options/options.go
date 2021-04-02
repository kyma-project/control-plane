package options

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type Options struct {
	KEBPollWaitDuration time.Duration
	KEBReqTimeout       time.Duration
	KEBRuntimeURL       *url.URL
	GardenerSecretPath  string
	GardenerNamespace   string
	ScrapeInterval      time.Duration
	WorkerPoolSize      int
	DebugPort           int
	ListenAddr          int
	LogLevel            logrus.Level
}

func ParseArgs() *Options {
	gardenerSecretPath := flag.String("gardener-secret-path", "/gardener/kubeconfig", "The path to the secret which contains kubeconfig of the Gardener MPS cluster")
	gardenerNamespace := flag.String("gardener-namespace", "garden-kyma-dev", "The namespace in gardener cluster where information about Kyma clusters are")
	scrapeInterval := flag.Duration("scrape-interval", 3*time.Minute, "The wait duration of the interval between 2 executions of metrics generation")
	workerPoolSize := flag.Int("worker-pool-size", 5, "The number of workers in the pool")
	logLevelStr := flag.String("log-level", "info", "The log-level of the application. E.g. fatal, error, info, debug etc")
	listenAddr := flag.Int("listen-addr", 8080, "The application starts server in this port to serve the metrics and healthz endpoints")
	debugPort := flag.Int("debug-port", 0, "The custom port to debug when needed")
	flag.Parse()

	logLevel, err := logrus.ParseLevel(*logLevelStr)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", logLevel)
	}

	return &Options{
		GardenerSecretPath: *gardenerSecretPath,
		GardenerNamespace:  *gardenerNamespace,
		ScrapeInterval:     *scrapeInterval,
		WorkerPoolSize:     *workerPoolSize,
		DebugPort:          *debugPort,
		LogLevel:           logLevel,
		ListenAddr:         *listenAddr,
	}
}

func (o *Options) String() string {
	return fmt.Sprintf("--gardener-secret-path=%s --gardener-namespace=%s --scrape-interval=%v "+
		"--worker-pool-size=%d --log-level=%s --listen-addr=%d, --debug-port=%d",
		o.GardenerSecretPath, o.GardenerNamespace, o.ScrapeInterval,
		o.WorkerPoolSize, o.LogLevel, o.ListenAddr, o.DebugPort)
}
