package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/alecthomas/kong"
	"github.com/kyma-project/control-plane/components/metris/internal/edp"
	"github.com/kyma-project/control-plane/components/metris/internal/gardener"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/kyma-project/control-plane/components/metris/internal/provider"
	"github.com/kyma-project/control-plane/components/metris/internal/service"
	"github.com/kyma-project/control-plane/components/metris/internal/tracing"
	"github.com/kyma-project/control-plane/components/metris/internal/utils"
	"github.com/kyma-project/control-plane/components/metris/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// import to initialize provider.
	_ "github.com/kyma-project/control-plane/components/metris/internal/provider/azure"
)

type cli struct {
	EDPConfig      edp.Config       `kong:"embed=true,prefix='edp-'"`
	ProviderType   string           `kong:"help='Provider to fetch metrics from. (${providers})',enum='${providers}',env='PROVIDER_TYPE',required=true,default='az',hidden=true"`
	ProviderConfig provider.Config  `kong:"embed=true,prefix='provider-'"`
	ListenAddr     string           `kong:"help='Address and port the metrics and health HTTP endpoints will bind to.',optional=true,env='METRIS_LISTEN_ADDRESS'"`
	DebugPort      int              `kong:"help='Port the debug HTTP endpoint will bind to. Always listen on localhost.',optional=true,env='METRIS_DEBUG_PORT'"`
	Tracing        tracing.Config   `kong:"embed=true"`
	ConfigFile     kong.ConfigFlag  `kong:"help='Location of the config file.',type='path'"`
	Kubeconfig     string           `kong:"help='Path to the Gardener kubeconfig file.',required=true,env='METRIS_KUBECONFIG'"`
	LogLevel       string           `kong:"help='Logging level. (${loglevels})',default='info',env='METRIS_LOGLEVEL'"`
	Version        kong.VersionFlag `kong:"help='Print version information and quit.'"`
}

func main() {
	app := cli{}
	_ = kong.Parse(&app,
		kong.Name("metris"),
		kong.Description("Metris is a metering component that collects data and sends them to EDP."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Vars{
			"version":   version.Print(),
			"loglevels": "debug,info,warn,error",
			"providers": "az",
		},
		kong.Configuration(kong.JSON, ""),
	)

	log.SetLogLevel(app.LogLevel)

	if app.Tracing.Enable {
		tc, err := tracing.New(app.Tracing)
		if err != nil {
			log.Panic(err)
		}

		defer tc.Stop()

		log.Named("zipkin").With("address", app.Tracing.ZipkinURL).Info("tracing enabled")
	}

	var (
		g              service.Workgroup
		clusterChannel = make(chan *gardener.Cluster, app.ProviderConfig.Buffer)
		eventChannel   = make(chan *edp.Event, app.EDPConfig.Buffer)
	)

	// start edp event handler
	edpclient := edp.NewClient(&app.EDPConfig, nil, eventChannel, log.Named("edp"))
	g.AddWithContext(edpclient.Start)

	// start provider to fetch metrics from the clusters
	app.ProviderConfig.ClusterChannel = clusterChannel
	app.ProviderConfig.EventsChannel = eventChannel
	app.ProviderConfig.Logger = log.Named(app.ProviderType)

	pro, err := provider.NewProvider(app.ProviderType, &app.ProviderConfig)
	if err != nil {
		log.Panic(err)
	}

	g.AddWithContext(pro.Run)

	// start gardener controller to sync clusters with provider
	gclient, err := gardener.NewClient(app.Kubeconfig)
	if err != nil {
		log.Panic(err)
	}

	ctrl, err := gardener.NewController(gclient, app.ProviderType, clusterChannel, log.Named("gardener"))
	if err != nil {
		log.Panic(err)
	}

	g.Add(ctrl.Run)

	// add metrics and health service.
	if len(app.ListenAddr) > 0 {
		metrissvc := service.Server{
			Addr:     app.ListenAddr,
			Logger:   log.Named("metris"),
			ServeMux: http.ServeMux{},
		}

		metrissvc.ServeMux.Handle("/metrics", promhttp.Handler())
		metrissvc.ServeMux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		metrissvc.ServeMux.HandleFunc("/logz", log.LevelHandler)
		// set loglevel: curl -X PUT -d '{"level":"debug"}' http://127.0.0.1:8080/logz

		g.Add(metrissvc.Start)
	}

	// add debug service.
	if app.DebugPort > 0 {
		// for security reason we always listen on localhost
		debugsvc := service.Server{
			Addr:     fmt.Sprintf("127.0.0.1:%d", app.DebugPort),
			Logger:   log.Named("debug"),
			ServeMux: http.ServeMux{},
		}

		debugsvc.ServeMux.HandleFunc("/debug/pprof/", pprof.Index)
		debugsvc.ServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		debugsvc.ServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		debugsvc.ServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		debugsvc.ServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		debugsvc.ServeMux.Handle("/debug/pprof/block", pprof.Handler("block"))
		debugsvc.ServeMux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		debugsvc.ServeMux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		debugsvc.ServeMux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		g.Add(debugsvc.Start)
	}

	// add exit handler.
	g.Add(utils.ExitHandler)

	err = g.Run()
	if err != nil {
		panic(err)
	}

	log.Info("metris stopped")
	log.Flush()
}
