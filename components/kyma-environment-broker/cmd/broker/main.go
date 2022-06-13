package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	gruntime "runtime"
	"runtime/pprof"
	"sort"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/health"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/kubeconfig"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/manager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/suspension"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/swagger"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/vrischmann/envconfig"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// Config holds configuration for the whole application
type Config struct {
	// DbInMemory allows to use memory storage instead of the postgres one.
	// Suitable for development purposes.
	DbInMemory bool `envconfig:"default=false"`

	// DisableProcessOperationsInProgress allows to disable processing operations
	// which are in progress on starting application. Set to true if you are
	// running in a separate testing deployment but with the production DB.
	DisableProcessOperationsInProgress bool `envconfig:"default=false"`

	// DevelopmentMode if set to true then errors are returned in http
	// responses, otherwise errors are only logged and generic message
	// is returned to client.
	// Currently works only with /info endpoints.
	DevelopmentMode bool `envconfig:"default=false"`

	// DumpProvisionerRequests enables dumping Provisioner requests. Must be disabled on Production environments
	// because some data must not be visible in the log file.
	DumpProvisionerRequests bool `envconfig:"default=false"`

	// OperationTimeout is used to check on a top-level if any operation didn't exceed the time for processing.
	// It is used for provisioning and deprovisioning operations.
	OperationTimeout time.Duration `envconfig:"default=24h"`

	Host       string `envconfig:"optional"`
	Port       string `envconfig:"default=8080"`
	StatusPort string `envconfig:"default=8071"`

	Provisioner input.Config
	Reconciler  reconciler.Config
	Director    director.Config
	Database    storage.Config
	Gardener    gardener.Config
	Kubeconfig  kubeconfig.Config

	ServiceManager servicemanager.Config

	KymaVersion                                string
	EnableOnDemandVersion                      bool `envconfig:"default=false"`
	ManagedRuntimeComponentsYAMLFilePath       string
	NewAdditionalRuntimeComponentsYAMLFilePath string
	SkrOidcDefaultValuesYAMLFilePath           string
	SkrDnsProvidersValuesYAMLFilePath          string
	DefaultRequestRegion                       string `envconfig:"default=cf-eu10"`
	UpdateProcessingEnabled                    bool   `envconfig:"default=false"`
	EnableBTPOperatorMigration                 bool   `envconfig:"default=true"`
	UpdateSubAccountMovementEnabled            bool   `envconfig:"default=false"`

	Broker          broker.Config
	CatalogFilePath string

	Avs avs.Config
	IAS ias.Config
	EDP edp.Config

	AuditLog auditlog.Config

	Notification notification.Config

	VersionConfig struct {
		Namespace string
		Name      string
	}

	KymaDashboardConfig dashboard.Config

	OrchestrationConfig orchestration.Config

	TrialRegionMappingFilePath string
	MaxPaginationPage          int `envconfig:"default=100"`

	LogLevel string `envconfig:"default=info"`

	// FreemiumProviders is a list of providers for freemium
	FreemiumProviders []string `envconfig:"default=aws"`

	DomainName string

	// Enable/disable profiler configuration. The profiler samples will be stored
	// under /tmp/profiler directory. Based on the deployment strategy, this will be
	// either ephemeral container filesystem or persistent storage
	Profiler ProfilerConfig
}

type ProfilerConfig struct {
	Path     string        `envconfig:"default=/tmp/profiler"`
	Sampling time.Duration `envconfig:"default=1s"`
	Memory   bool
}

const (
	createRuntimeStageName = "create_runtime"
	checkRuntimeStageName  = "check_runtime"
	createKymaStageName    = "create_kyma"
	checkKymaStageName     = "check_kyma"
	startStageName         = "start"
)

func periodicProfile(logger lager.Logger, profiler ProfilerConfig) {
	if profiler.Memory == false {
		return
	}
	logger.Info(fmt.Sprintf("Starting periodic profiler %v", profiler))
	if err := os.MkdirAll(profiler.Path, os.ModePerm); err != nil {
		logger.Error(fmt.Sprintf("Failed to create dir %v for profile storage", profiler.Path), err)
	}
	for {
		profName := fmt.Sprintf("%v/mem-%v.pprof", profiler.Path, time.Now().Unix())
		logger.Info(fmt.Sprintf("Creating periodic memory profile %v", profName))
		profFile, err := os.Create(profName)
		if err != nil {
			logger.Error(fmt.Sprintf("Creating periodic memory profile %v failed", profName), err)
		}
		pprof.Lookup("allocs").WriteTo(profFile, 0)
		gruntime.GC()
		time.Sleep(profiler.Sampling)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	// check default Kyma versions
	err = checkDefaultVersions(cfg.KymaVersion)
	panicOnError(err)

	cfg.OrchestrationConfig.KymaVersion = cfg.KymaVersion
	cfg.OrchestrationConfig.KubernetesVersion = cfg.Provisioner.KubernetesVersion

	// create logger
	logger := lager.NewLogger("kyma-env-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	logger.Info("Starting Kyma Environment Broker")

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})
	if cfg.LogLevel != "" {
		l, _ := logrus.ParseLevel(cfg.LogLevel)
		logs.SetLevel(l)
	}

	logger.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Host, cfg.StatusPort, logs).ServeAsync()
	go periodicProfile(logger, cfg.Profiler)

	// create provisioner client
	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioner.URL, cfg.DumpProvisionerRequests)

	reconcilerClient := reconciler.NewReconcilerClient(http.DefaultClient, logs.WithField("service", "reconciler"), &cfg.Reconciler)

	// create kubernetes client
	k8sCfg, err := config.GetConfig()
	fatalOnError(err)
	cli, err := initClient(k8sCfg)
	fatalOnError(err)

	// create director client
	directorClient := director.NewDirectorClient(ctx, cfg.Director, logs.WithField("service", "directorClient"))

	// create storage
	cipher := storage.NewEncrypter(cfg.Database.SecretKey)
	var db storage.BrokerStorage
	if cfg.DbInMemory {
		db = storage.NewMemoryStorage()
	} else {
		store, conn, err := storage.NewFromConfig(cfg.Database, cipher, logs.WithField("service", "storage"))
		fatalOnError(err)
		db = store
		dbStatsCollector := sqlstats.NewStatsCollector("broker", conn)
		prometheus.MustRegister(dbStatsCollector)
	}

	// Auditlog
	fileSystem := afero.NewOsFs()

	// Customer Notification
	clientHTTPForNotification := httputil.NewClient(60, true)
	notificationClient := notification.NewClient(clientHTTPForNotification, notification.ClientConfig{
		URL: cfg.Notification.Url,
	})
	notificationBuilder := notification.NewBundleBuilder(notificationClient, cfg.Notification)

	// Register disabler. Convention:
	// {component-name} : {component-disabler-service}
	//
	// Using map is intentional - we ensure that component name is not duplicated.
	optionalComponentsDisablers := runtime.ComponentsDisablers{
		components.Kiali:   runtime.NewGenericComponentDisabler(components.Kiali),
		components.Tracing: runtime.NewGenericComponentDisabler(components.Tracing),
	}
	optComponentsSvc := runtime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := runtime.NewDisabledComponentsProvider()

	// switch for runtime.NewComponentsProvider() when ready
	componentsProvider := runtime.NewComponentsListProvider(cfg.ManagedRuntimeComponentsYAMLFilePath, cfg.NewAdditionalRuntimeComponentsYAMLFilePath)
	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err)
	cfg.Gardener.DNSProviders, err = gardener.ReadDNSProvidersValuesFromYAML(cfg.SkrDnsProvidersValuesYAMLFilePath)
	fatalOnError(err)
	dynamicGardener, err := dynamic.NewForConfig(gardenerClusterConfig)
	fatalOnError(err)

	gardenerNamespace := fmt.Sprintf("garden-%v", cfg.Gardener.Project)
	gardenerAccountPool := hyperscaler.NewAccountPool(dynamicGardener, gardenerNamespace)
	gardenerSharedPool := hyperscaler.NewSharedGardenerAccountPool(dynamicGardener, gardenerNamespace)
	accountProvider := hyperscaler.NewAccountProvider(gardenerAccountPool, gardenerSharedPool)

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	fatalOnError(err)
	logs.Infof("Platform region mapping for trial: %v", regions)
	oidcDefaultValues, err := runtime.ReadOIDCDefaultValuesFromYAML(cfg.SkrOidcDefaultValuesYAMLFilePath)
	fatalOnError(err)
	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentsProvider,
		cfg.Provisioner, cfg.KymaVersion, regions, cfg.FreemiumProviders, oidcDefaultValues)
	fatalOnError(err)

	edpClient := edp.NewClient(cfg.EDP, logs.WithField("service", "edpClient"))

	avsClient, err := avs.NewClient(ctx, cfg.Avs, logs)
	fatalOnError(err)
	avsDel := avs.NewDelegator(avsClient, cfg.Avs, db.Operations())
	externalEvalAssistant := avs.NewExternalEvalAssistant(cfg.Avs)
	internalEvalAssistant := avs.NewInternalEvalAssistant(cfg.Avs)
	externalEvalCreator := provisioning.NewExternalEvalCreator(avsDel, cfg.Avs.Disabled, externalEvalAssistant)
	internalEvalUpdater := provisioning.NewInternalEvalUpdater(avsDel, internalEvalAssistant, cfg.Avs)
	upgradeEvalManager := avs.NewEvaluationManager(avsDel, cfg.Avs)

	// IAS
	clientHTTPForIAS := httputil.NewClient(60, cfg.IAS.SkipCertVerification)
	if cfg.IAS.TLSRenegotiationEnable {
		clientHTTPForIAS = httputil.NewRenegotiationTLSClient(30, cfg.IAS.SkipCertVerification)
	}
	iasClient := ias.NewClient(clientHTTPForIAS, ias.ClientConfig{
		URL:    cfg.IAS.URL,
		ID:     cfg.IAS.UserID,
		Secret: cfg.IAS.UserSecret,
	})
	bundleBuilder := ias.NewBundleBuilder(iasClient, cfg.IAS)

	// application event broker
	eventBroker := event.NewPubSub(logs)

	// metrics collectors
	metrics.RegisterAll(eventBroker, db.Operations(), db.Instances())

	//setup runtime overrides appender
	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)

	serviceManagerClientFactory := servicemanager.NewClientFactory(cfg.ServiceManager)

	// define steps
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, accountVersionMapping, db.RuntimeStates())

	// run queues
	const workersAmount = 5
	provisionManager := provisioning.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs.WithField("provisioning", "manager"))
	provisionQueue := NewProvisioningProcessingQueue(ctx, provisionManager, 60, &cfg, db, provisionerClient, directorClient, inputFactory,
		avsDel, internalEvalAssistant, externalEvalCreator, internalEvalUpdater, runtimeVerConfigurator,
		runtimeOverrides, serviceManagerClientFactory, bundleBuilder,
		edpClient, accountProvider, fileSystem, reconcilerClient, logs)

	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))
	deprovisionQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, &cfg, db, eventBroker, provisionerClient,
		avsDel, internalEvalAssistant, externalEvalAssistant, serviceManagerClientFactory, bundleBuilder, edpClient, accountProvider, reconcilerClient,
		k8sClientProvider, logs)

	updateManager := update.NewManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs)
	updateQueue := NewUpdateProcessingQueue(ctx, updateManager, 20, db, inputFactory, provisionerClient, eventBroker,
		runtimeVerConfigurator, db.RuntimeStates(), componentsProvider, reconcilerClient, cfg, k8sClientProvider, logs)

	/***/
	servicesConfig, err := broker.NewServicesConfigFromFile(cfg.CatalogFilePath)
	fatalOnError(err)

	// create server
	router := mux.NewRouter()

	createAPI(router, servicesConfig, inputFactory, &cfg, db, provisionQueue, deprovisionQueue, updateQueue, logger, logs, inputFactory.GetPlanDefaults)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// create SKR kubeconfig endpoint
	kcBuilder := kubeconfig.NewBuilder(provisionerClient)
	kcHandler := kubeconfig.NewHandler(db, kcBuilder, cfg.Kubeconfig.AllowOrigins, logs.WithField("service", "kubeconfigHandle"))
	kcHandler.AttachRoutes(router)

	runtimeLister := orchestration.NewRuntimeLister(db.Instances(), db.Operations(), runtime.NewConverter(cfg.DefaultRequestRegion), logs)
	runtimeResolver := orchestrationExt.NewGardenerRuntimeResolver(dynamicGardener, gardenerNamespace, runtimeLister, logs)

	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, nil, time.Minute, runtimeVerConfigurator, runtimeResolver, upgradeEvalManager,
		&cfg, internalEvalAssistant, reconcilerClient, serviceManagerClientFactory, notificationBuilder, fileSystem, logs, cli, 1)
	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory,
		nil, time.Minute, runtimeResolver, upgradeEvalManager, notificationBuilder, logs, cli, cfg, 1)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, clusterQueue, cfg.MaxPaginationPage, logs)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(internal.OperationTypeProvision, db.Operations(), provisionQueue, logs)
		fatalOnError(err)
		err = processOperationsInProgressByType(internal.OperationTypeDeprovision, db.Operations(), deprovisionQueue, logs)
		fatalOnError(err)
		err = processOperationsInProgressByType(internal.OperationTypeUpdate, db.Operations(), updateQueue, logs)
		fatalOnError(err)
		err = reprocessOrchestrations(orchestrationExt.UpgradeKymaOrchestration, db.Orchestrations(), db.Operations(), kymaQueue, logs)
		fatalOnError(err)
		err = reprocessOrchestrations(orchestrationExt.UpgradeClusterOrchestration, db.Orchestrations(), db.Operations(), clusterQueue, logs)
		fatalOnError(err)
	} else {
		logger.Info("Skipping processing operation in progress on start")
	}

	// configure templates e.g. {{.domain}} to replace it with the domain name
	swaggerTemplates := map[string]string{
		"domain": cfg.DomainName,
	}
	err = swagger.NewTemplate("/swagger", swaggerTemplates).Execute()
	fatalOnError(err)

	// create /orchestration
	orchestrationHandler.AttachRoutes(router)

	// create list runtimes endpoint
	runtimeHandler := runtime.NewHandler(db.Instances(), db.Operations(), db.RuntimeStates(), cfg.MaxPaginationPage, cfg.DefaultRequestRegion)
	runtimeHandler.AttachRoutes(router)

	router.StrictSlash(true).PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("/swagger"))))
	svr := handlers.CustomLoggingHandler(os.Stdout, router, func(writer io.Writer, params handlers.LogFormatterParams) {
		logs.Infof("Call handled: method=%s url=%s statusCode=%d size=%d", params.Request.Method, params.URL.Path, params.StatusCode, params.Size)
	})

	fatalOnError(http.ListenAndServe(cfg.Host+":"+cfg.Port, svr))
}

func k8sClientProvider(kcfg string) (client.Client, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kcfg))
	if err != nil {
		return nil, err
	}

	sch := runtime2.NewScheme()
	apiextensionsv1.AddToScheme(sch)

	k8sCli, err := client.New(restCfg, client.Options{
		Scheme: sch,
	})
	return k8sCli, err
}

func checkDefaultVersions(versions ...string) error {
	for _, version := range versions {
		if !isVersionFollowingSemanticVersioning(version) {
			return errors.New("Kyma default versions are not following semantic versioning")
		}
	}
	return nil
}

func isVersionFollowingSemanticVersioning(version string) bool {
	regexpToMatch := regexp.MustCompile("(^[0-9]+\\.{1}).*")
	if regexpToMatch.MatchString(version) {
		return true
	}
	return false
}

func createAPI(router *mux.Router, servicesConfig broker.ServicesConfig, planValidator broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisionQueue, deprovisionQueue, updateQueue *process.Queue, logger lager.Logger, logs logrus.FieldLogger, planDefaults broker.PlanDefaults) {
	suspensionCtxHandler := suspension.NewContextUpdateHandler(db.Operations(), provisionQueue, deprovisionQueue, logs)

	defaultPlansConfig, err := servicesConfig.DefaultPlansConfig()
	fatalOnError(err)

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		broker.NewServices(cfg.Broker, servicesConfig, logs),
		broker.NewProvision(cfg.Broker, cfg.Gardener, db.Operations(), db.Instances(), provisionQueue, planValidator, defaultPlansConfig, cfg.EnableOnDemandVersion, planDefaults, logs, cfg.KymaDashboardConfig),
		broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		broker.NewUpdate(cfg.Broker, db.Instances(), db.RuntimeStates(), db.Operations(), suspensionCtxHandler, cfg.UpdateProcessingEnabled, cfg.UpdateSubAccountMovementEnabled, updateQueue, planDefaults, logs, cfg.KymaDashboardConfig),
		broker.NewGetInstance(cfg.Broker, db.Instances(), db.Operations(), logs),
		broker.NewLastOperation(db.Operations(), logs),
		broker.NewBind(logs),
		broker.NewUnbind(logs),
		broker.NewGetBinding(logs),
		broker.NewLastBindingOperation(logs),
	}

	router.Use(middleware.AddRegionToContext(cfg.DefaultRequestRegion))
	router.Use(middleware.AddProviderToContext())
	for _, prefix := range []string{
		"/oauth/",          // oauth2 handled by Ory
		"/oauth/{region}/", // oauth2 handled by Ory with region
	} {
		route := router.PathPrefix(prefix).Subrouter()
		broker.AttachRoutes(route, kymaEnvBroker, logger)
	}

	respWriter := httputil.NewResponseWriter(logs, cfg.DevelopmentMode)
	runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(db.Instances(), db.Operations(), defaultPlansConfig, cfg.DefaultRequestRegion, respWriter)
	router.Handle("/info/runtimes", runtimesInfoHandler)
}

// queues all in progress operations by type
func processOperationsInProgressByType(opType internal.OperationType, op storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	operations, err := op.GetNotFinishedOperationsByType(opType)
	if err != nil {
		return errors.Wrap(err, "while getting in progress operations from storage")
	}
	for _, operation := range operations {
		queue.Add(operation.ID)
		log.Infof("Resuming the processing of %s operation ID: %s", opType, operation.ID)
	}
	return nil
}

func reprocessOrchestrations(orchestrationType orchestrationExt.Type, orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	if err := processCancelingOrchestrations(orchestrationType, orchestrationsStorage, operationsStorage, queue, log); err != nil {
		return errors.Wrapf(err, "while processing canceled %s orchestrations", orchestrationType)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.InProgress, orchestrationsStorage, queue, log); err != nil {
		return errors.Wrapf(err, "while processing in progress %s orchestrations", orchestrationType)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.Pending, orchestrationsStorage, queue, log); err != nil {
		return errors.Wrapf(err, "while processing pending %s orchestrations", orchestrationType)
	}
	if err := processOrchestration(orchestrationType, orchestrationExt.Retrying, orchestrationsStorage, queue, log); err != nil {
		return errors.Wrapf(err, "while processing retrying %s orchestrations", orchestrationType)
	}
	return nil
}

func processOrchestration(orchestrationType orchestrationExt.Type, state string, orchestrationsStorage storage.Orchestrations, queue *process.Queue, log logrus.FieldLogger) error {
	filter := dbmodel.OrchestrationFilter{
		Types:  []string{string(orchestrationType)},
		States: []string{state},
	}
	orchestrations, _, _, err := orchestrationsStorage.List(filter)
	if err != nil {
		return errors.Wrapf(err, "while getting %s %s orchestrations from storage", state, orchestrationType)
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		queue.Add(o.OrchestrationID)
		log.Infof("Resuming the processing of %s %s orchestration ID: %s", state, orchestrationType, o.OrchestrationID)
	}
	return nil
}

// processCancelingOrchestrations reprocess orchestrations with canceling state only when some in progress operations exists
// reprocess only one orchestration to not clog up the orchestration queue on start
func processCancelingOrchestrations(orchestrationType orchestrationExt.Type, orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	filter := dbmodel.OrchestrationFilter{
		Types:  []string{string(orchestrationType)},
		States: []string{orchestrationExt.Canceling},
	}
	orchestrations, _, _, err := orchestrationsStorage.List(filter)
	if err != nil {
		return errors.Wrapf(err, "while getting canceling %s orchestrations from storage", orchestrationType)
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		count := 0
		err = nil
		if orchestrationType == orchestrationExt.UpgradeKymaOrchestration {
			_, count, _, err = operationsStorage.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestrationExt.InProgress}})
		} else if orchestrationType == orchestrationExt.UpgradeClusterOrchestration {
			_, count, _, err = operationsStorage.ListUpgradeClusterOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestrationExt.InProgress}})
		}
		if err != nil {
			return errors.Wrapf(err, "while listing %s operations for orchestration %s", orchestrationType, o.OrchestrationID)
		}

		if count > 0 {
			log.Infof("Resuming the processing of %s %s orchestration ID: %s", orchestrationExt.Canceling, orchestrationType, o.OrchestrationID)
			queue.Add(o.OrchestrationID)
			return nil
		}
	}
	return nil
}

func initClient(cfg *rest.Config) (client.Client, error) {
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(cfg)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, errors.Wrap(err, "while waiting for client mapper")
		}
	}
	cli, err := client.New(cfg, client.Options{Mapper: mapper})
	if err != nil {
		return nil, errors.Wrap(err, "while creating a client")
	}
	return cli, nil
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewProvisioningProcessingQueue(ctx context.Context, provisionManager *provisioning.StagedManager, workersAmount int,
	cfg *Config, db storage.BrokerStorage, provisionerClient provisioner.Client, directorClient provisioning.DirectorClient,
	inputFactory input.CreatorForPlan, avsDel *avs.Delegator, internalEvalAssistant *avs.InternalEvalAssistant,
	externalEvalCreator *provisioning.ExternalEvalCreator, internalEvalUpdater *provisioning.InternalEvalUpdater,
	runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator, runtimeOverrides provisioning.RuntimeOverridesAppender,
	smcf provisioning.SMClientFactory, bundleBuilder ias.BundleBuilder, edpClient provisioning.EDPClient,
	accountProvider hyperscaler.AccountProvider, fileSystem afero.Fs, reconcilerClient reconciler.Client, logs logrus.FieldLogger) *process.Queue {

	const postActionsStageName = "post_actions"
	provisionManager.DefineStages([]string{startStageName, createRuntimeStageName,
		checkKymaStageName, postActionsStageName})
	/*
			The provisioning process contains the following stages:
			1. "start" - changes the state from pending to in progress if no deprovisioning is ongoing.
			2. "create_runtime" - collects all information needed to make an input for the Provisioner request as overrides and labels.
			Those data is collected using an InputCreator which is not persisted. That's why all steps which prepares such data must be in the same stage as "create runtime step".
		    All steps which requires InputCreator must be run in this stage.
			3. "check_kyma" - checks if the Kyma is installed
			4. "post_actions" - all steps which must be executed after the runtime is provisioned

			Once the stage is done it will never be retried.
	*/

	provisioningSteps := []struct {
		disabled  bool
		stage     string
		step      provisioning.Step
		condition provisioning.StepCondition
	}{
		{
			stage: startStageName,
			step:  provisioning.NewStartStep(db.Operations(), db.Instances()),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewInitialisationStep(db.Operations(), db.Instances(), inputFactory, cfg.Provisioner.ProvisioningTimeout, cfg.OperationTimeout, runtimeVerConfigurator, smcf),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider),
		},
		{
			stage:    createRuntimeStageName,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
		{
			stage:    createRuntimeStageName,
			step:     provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
		},
		{
			condition: provisioning.WhenBTPOperatorCredentialsProvided,
			stage:     createRuntimeStageName,
			step:      provisioning.NewBTPOperatorOverridesStep(db.Operations()),
		},
		{
			condition: provisioning.ForKyma1,
			stage:     createRuntimeStageName,
			step:      provisioning.NewAuditLogOverridesStep(fileSystem, db.Operations(), cfg.AuditLog),
		},
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewBusolaMigratorOverridesStep(),
		},
		{
			condition: provisioning.ForKyma1,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeStep(db.Operations(), db.RuntimeStates(), db.Instances(), provisionerClient),
		},
		{
			condition: provisioning.ForKyma2,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateRuntimeWithoutKymaStep(db.Operations(), db.RuntimeStates(), db.Instances(), provisionerClient),
		},
		// check the runtime status
		{
			stage: createRuntimeStageName,
			step:  provisioning.NewCheckRuntimeStep(db.Operations(), provisionerClient, cfg.Provisioner.ProvisioningTimeout),
		},
		{
			condition: provisioning.ForKyma2,
			stage:     createRuntimeStageName,
			step:      provisioning.NewGetKubeconfigStep(db.Operations(), provisionerClient),
		},
		{
			condition: provisioning.ForKyma2,
			stage:     createRuntimeStageName,
			step:      provisioning.NewCreateClusterConfiguration(db.Operations(), db.RuntimeStates(), reconcilerClient),
		},
		{
			condition: provisioning.ForKyma2,
			stage:     checkKymaStageName,
			step:      provisioning.NewCheckClusterConfigurationStep(db.Operations(), reconcilerClient, cfg.Reconciler.ProvisioningTimeout),
		},
		// post actions
		{
			stage: postActionsStageName,
			step:  provisioning.NewExternalEvalStep(externalEvalCreator),
		},
		{
			stage: postActionsStageName,
			step:  provisioning.NewRuntimeTagsStep(internalEvalUpdater, provisionerClient),
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			err := provisionManager.AddStep(step.stage, step.step, step.condition)
			if err != nil {
				fatalOnError(err)
			}
		}
	}

	queue := process.NewQueue(provisionManager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}

func NewUpdateProcessingQueue(ctx context.Context, manager *update.Manager, workersAmount int, db storage.BrokerStorage, inputFactory input.CreatorForPlan,
	provisionerClient provisioner.Client, publisher event.Publisher, runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator, runtimeStatesDb storage.RuntimeStates,
	runtimeProvider input.ComponentListProvider, reconcilerClient reconciler.Client, cfg Config, k8sClientProvider func(kcfg string) (client.Client, error), logs logrus.FieldLogger) *process.Queue {

	ifBTPMigrationEnabled := func(c update.StepCondition) update.StepCondition {
		if cfg.EnableBTPOperatorMigration {
			return c
		}
		return func(o internal.UpdatingOperation) bool {
			return false
		}
	}
	negation := func(c update.StepCondition) update.StepCondition {
		return func(o internal.UpdatingOperation) bool {
			v := c(o)
			return !v
		}
	}

	btpMigrationEnabled := func(o internal.UpdatingOperation) bool {
		return cfg.EnableBTPOperatorMigration
	}

	manager.DefineStages([]string{"cluster", "migration", "migration-check", "remove-sc-migration", "remove-sc-migration-check", "check"})
	updateSteps := []struct {
		stage     string
		step      update.Step
		condition update.StepCondition
	}{
		{
			stage: "cluster",
			step:  update.NewInitialisationStep(db.Instances(), db.Operations(), inputFactory),
		},
		{
			stage:     "cluster",
			step:      update.NewUpgradeShootStep(db.Operations(), db.RuntimeStates(), provisionerClient),
			condition: negation(ifBTPMigrationEnabled(update.ForMigration)),
		},
		{
			stage:     "migration",
			step:      update.NewInitKymaVersionStep(db.Operations(), runtimeVerConfigurator, runtimeStatesDb),
			condition: btpMigrationEnabled,
		},
		{
			stage:     "migration",
			step:      update.NewGetKubeconfigStep(db.Operations(), provisionerClient, k8sClientProvider),
			condition: ifBTPMigrationEnabled(update.ForBTPOperatorCredentialsProvided),
		},
		{
			stage:     "migration",
			step:      update.NewBTPOperatorCheckStep(db.Operations()),
			condition: ifBTPMigrationEnabled(update.ForBTPOperatorCredentialsProvided),
		},
		{
			stage:     "migration",
			step:      update.NewBTPOperatorOverridesStep(db.Operations(), runtimeProvider),
			condition: ifBTPMigrationEnabled(update.ForBTPOperatorCredentialsProvided),
		},
		{
			stage:     "migration",
			step:      update.NewSCMigrationStep(db.Operations(), runtimeProvider),
			condition: ifBTPMigrationEnabled(update.ForMigration),
		},
		{
			stage:     "migration",
			step:      update.NewApplyReconcilerConfigurationStep(db.Operations(), db.RuntimeStates(), reconcilerClient),
			condition: ifBTPMigrationEnabled(update.RequiresReconcilerUpdate),
		},
		{
			stage:     "migration-check",
			step:      update.NewCheckReconcilerState(db.Operations(), reconcilerClient),
			condition: ifBTPMigrationEnabled(update.CheckReconcilerStatus),
		},
		{
			stage:     "remove-sc-migration",
			step:      update.NewInitKymaVersionStep(db.Operations(), runtimeVerConfigurator, runtimeStatesDb),
			condition: btpMigrationEnabled,
		},
		{
			stage:     "remove-sc-migration",
			step:      update.NewGetKubeconfigStep(db.Operations(), provisionerClient, k8sClientProvider),
			condition: ifBTPMigrationEnabled(update.ForBTPOperatorCredentialsProvided),
		},
		{
			stage:     "remove-sc-migration",
			step:      update.NewSCMigrationFinalizationStep(reconcilerClient),
			condition: ifBTPMigrationEnabled(update.ForMigration),
		},
		{
			stage:     "remove-sc-migration",
			step:      update.NewApplyReconcilerConfigurationStep(db.Operations(), db.RuntimeStates(), reconcilerClient),
			condition: ifBTPMigrationEnabled(update.RequiresReconcilerUpdateForMigration),
		},
		{
			stage:     "remove-sc-migration-check",
			step:      update.NewCheckReconcilerState(db.Operations(), reconcilerClient),
			condition: ifBTPMigrationEnabled(update.CheckReconcilerStatus),
		},
		{
			stage:     "check",
			step:      update.NewCheckStep(db.Operations(), provisionerClient, 40*time.Minute),
			condition: negation(ifBTPMigrationEnabled(update.ForBTPOperatorCredentialsProvided)),
		},
	}

	for _, step := range updateSteps {
		err := manager.AddStep(step.stage, step.step, step.condition)
		if err != nil {
			fatalOnError(err)
		}
	}
	queue := process.NewQueue(manager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}

func NewDeprovisioningProcessingQueue(ctx context.Context, workersAmount int, deprovisionManager *deprovisioning.Manager, cfg *Config, db storage.BrokerStorage, pub event.Publisher,
	provisionerClient provisioner.Client, avsDel *avs.Delegator, internalEvalAssistant *avs.InternalEvalAssistant,
	externalEvalAssistant *avs.ExternalEvalAssistant, smcf deprovisioning.SMClientFactory, bundleBuilder ias.BundleBuilder,
	edpClient deprovisioning.EDPClient, accountProvider hyperscaler.AccountProvider, reconcilerClient reconciler.Client,
	k8sClientProvider func(kcfg string) (client.Client, error), logs logrus.FieldLogger) *process.Queue {

	deprovisioningInit := deprovisioning.NewInitialisationStep(db.Operations(), db.Instances(), provisionerClient, accountProvider, smcf, cfg.OperationTimeout)
	deprovisionManager.InitStep(deprovisioningInit)

	deprovisioningSteps := []struct {
		disabled bool
		weight   int
		step     deprovisioning.Step
	}{
		{
			weight: 1,
			step:   deprovisioning.NewGetKubeconfigStep(db.Operations(), provisionerClient, k8sClientProvider),
		},
		{
			weight: 1,
			step:   deprovisioning.NewRemoveServiceInstanceStep(db.Operations()),
		},
		{
			weight: 1,
			step:   deprovisioning.NewAvsEvaluationsRemovalStep(avsDel, db.Operations(), externalEvalAssistant, internalEvalAssistant),
		},
		{
			weight:   1,
			step:     deprovisioning.NewEDPDeregistrationStep(edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			weight:   1,
			step:     deprovisioning.NewIASDeregistrationStep(db.Operations(), bundleBuilder),
			disabled: cfg.IAS.Disabled,
		},
		{
			weight: 5,
			step:   deprovisioning.NewDeregisterClusterStep(db.Operations(), reconcilerClient),
		},
		{
			weight: 6,
			step:   deprovisioning.NewCheckClusterDeregistrationStep(db.Operations(), reconcilerClient, 90*time.Minute),
		},
		{
			weight: 10,
			step:   deprovisioning.NewRemoveRuntimeStep(db.Operations(), db.Instances(), provisionerClient, cfg.Provisioner.DeprovisioningTimeout),
		},
	}
	for _, step := range deprovisioningSteps {
		if !step.disabled {
			deprovisionManager.AddStep(step.weight, step.step)
		}
	}

	queue := process.NewQueue(deprovisionManager, logs)
	queue.Run(ctx.Done(), workersAmount)

	return queue
}

func NewKymaOrchestrationProcessingQueue(ctx context.Context, db storage.BrokerStorage,
	runtimeOverrides upgrade_kyma.RuntimeOverridesAppender, provisionerClient provisioner.Client,
	pub event.Publisher, inputFactory input.CreatorForPlan, icfg *upgrade_kyma.TimeSchedule,
	pollingInterval time.Duration, runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator,
	runtimeResolver orchestrationExt.RuntimeResolver, upgradeEvalManager *avs.EvaluationManager,
	cfg *Config, internalEvalAssistant *avs.InternalEvalAssistant, reconcilerClient reconciler.Client, smcf internal.SMClientFactory,
	notificationBuilder notification.BundleBuilder, fileSystem afero.Fs, logs logrus.FieldLogger, cli client.Client, speedFactor int) *process.Queue {

	upgradeKymaManager := upgrade_kyma.NewManager(db.Operations(), pub, logs.WithField("upgradeKyma", "manager"))
	upgradeKymaInit := upgrade_kyma.NewInitialisationStep(db.Operations(), db.Orchestrations(), db.Instances(),
		provisionerClient, inputFactory, upgradeEvalManager, icfg, runtimeVerConfigurator, smcf, notificationBuilder)

	upgradeKymaManager.InitStep(upgradeKymaInit)
	upgradeKymaSteps := []struct {
		disabled bool
		weight   int
		step     upgrade_kyma.Step
		cnd      upgrade_kyma.StepCondition
	}{
		// check cluster configuration is the first step - to not execute other steps, when cluster configuration was applied
		// this should be moved to the end when we introduce stages like in the provisioning process
		// (also return operation, 0, nil at the end of apply_cluster_configuration)
		{
			weight: 1,
			step:   upgrade_kyma.NewCheckClusterConfigurationStep(db.Operations(), reconcilerClient, upgradeEvalManager, cfg.Reconciler.ProvisioningTimeout),
			cnd:    upgrade_kyma.ForKyma2,
		},
		{
			weight: 3,
			cnd:    upgrade_kyma.WhenBTPOperatorCredentialsProvided,
			step:   upgrade_kyma.NewBTPOperatorOverridesStep(db.Operations()),
		},
		{
			weight: 4,
			step:   upgrade_kyma.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
		},
		{
			weight: 5,
			step:   upgrade_kyma.NewAuditLogOverridesStep(fileSystem, db.Operations(), cfg.AuditLog),
			cnd:    upgrade_kyma.ForKyma1,
		},
		{
			weight: 6,
			step:   upgrade_kyma.NewBusolaMigratorOverridesStep(),
		},
		{
			weight: 8,
			step:   upgrade_kyma.NewUpgradeKymaStep(db.Operations(), db.RuntimeStates(), provisionerClient, icfg),
			cnd:    upgrade_kyma.ForKyma1,
		},
		{
			weight:   8,
			step:     upgrade_kyma.NewSendNotificationStep(db.Operations(), notificationBuilder),
			disabled: cfg.Notification.Disabled,
		},
		{
			weight: 9,
			step:   upgrade_kyma.NewGetKubeconfigStep(db.Operations(), provisionerClient),
			cnd:    upgrade_kyma.ForKyma2,
		},
		{
			weight: 10,
			step:   upgrade_kyma.NewApplyClusterConfigurationStep(db.Operations(), db.RuntimeStates(), reconcilerClient),
			cnd:    upgrade_kyma.ForKyma2,
		},
	}
	for _, step := range upgradeKymaSteps {
		if !step.disabled {
			upgradeKymaManager.AddStep(step.weight, step.step, step.cnd)
		}
	}

	orchestrateKymaManager := manager.NewUpgradeKymaManager(db.Orchestrations(), db.Operations(), db.Instances(),
		upgradeKymaManager, runtimeResolver, pollingInterval, smcf, logs.WithField("upgradeKyma", "orchestration"),
		cli, &cfg.OrchestrationConfig, notificationBuilder, speedFactor)
	queue := process.NewQueue(orchestrateKymaManager, logs)

	queue.Run(ctx.Done(), 3)

	return queue
}

func NewClusterOrchestrationProcessingQueue(ctx context.Context, db storage.BrokerStorage, provisionerClient provisioner.Client,
	pub event.Publisher, inputFactory input.CreatorForPlan, icfg *upgrade_cluster.TimeSchedule, pollingInterval time.Duration,
	runtimeResolver orchestrationExt.RuntimeResolver, upgradeEvalManager *avs.EvaluationManager, notificationBuilder notification.BundleBuilder, logs logrus.FieldLogger,
	cli client.Client, cfg Config, speedFactor int) *process.Queue {

	upgradeClusterManager := upgrade_cluster.NewManager(db.Operations(), pub, logs.WithField("upgradeCluster", "manager"))
	upgradeClusterInit := upgrade_cluster.NewInitialisationStep(db.Operations(), db.Orchestrations(), provisionerClient, inputFactory, upgradeEvalManager, icfg, notificationBuilder)
	upgradeClusterManager.InitStep(upgradeClusterInit)

	upgradeClusterSteps := []struct {
		disabled bool
		weight   int
		step     upgrade_cluster.Step
	}{
		{
			weight:   10,
			step:     upgrade_cluster.NewSendNotificationStep(db.Operations(), notificationBuilder),
			disabled: cfg.Notification.Disabled,
		},
		{
			weight: 10,
			step:   upgrade_cluster.NewUpgradeClusterStep(db.Operations(), db.RuntimeStates(), provisionerClient, icfg),
		},
	}
	for _, step := range upgradeClusterSteps {
		if !step.disabled {
			upgradeClusterManager.AddStep(step.weight, step.step)
		}
	}

	orchestrateClusterManager := manager.NewUpgradeClusterManager(db.Orchestrations(), db.Operations(), db.Instances(),
		upgradeClusterManager, runtimeResolver, pollingInterval, logs.WithField("upgradeCluster", "orchestration"),
		cli, cfg.OrchestrationConfig, notificationBuilder, speedFactor)
	queue := process.NewQueue(orchestrateClusterManager, logs)

	queue.Run(ctx.Done(), 3)

	return queue
}
