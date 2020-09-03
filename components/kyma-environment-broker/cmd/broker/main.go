package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/director/oauth"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/health"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/hyperscaler/azure"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	gcli "github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Config holds configuration for the whole application
type Config struct {
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

	Host       string `envconfig:"optional"`
	Port       string `envconfig:"default=8080"`
	StatusPort string `envconfig:"default=8071"`

	Provisioning input.Config
	Director     director.Config
	Database     storage.Config
	Gardener     gardener.Config

	ServiceManager provisioning.ServiceManagerOverrideConfig

	KymaVersion                          string
	EnableOnDemandVersion                bool `envconfig:"default=false"`
	ManagedRuntimeComponentsYAMLFilePath string
	DefaultRequestRegion                 string `envconfig:"default=cf-eu10"`

	Broker broker.Config

	Avs avs.Config
	LMS lms.Config
	IAS ias.Config
	EDP edp.Config

	AuditLog auditlog.Config

	VersionConfig struct {
		Namespace string
		Name      string
	}

	TrialRegionMappingFilePath string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create and fill config
	var cfg Config
	err := envconfig.InitWithPrefix(&cfg, "APP")
	fatalOnError(err)

	// create logger
	logger := lager.NewLogger("kyma-env-broker")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	logger.Info("Starting Kyma Environment Broker")

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	logger.Info("Registering healthz endpoint for health probes")
	health.NewServer(cfg.Host, cfg.StatusPort, logs).ServeAsync()

	// create provisioner client
	provisionerClient := provisioner.NewProvisionerClient(cfg.Provisioning.URL, cfg.DumpProvisionerRequests)

	// create kubernetes client
	k8sCfg, err := config.GetConfig()
	fatalOnError(err)
	cli, err := initClient(k8sCfg)
	fatalOnError(err)

	// create director client on the base of graphQL client and OAuth client
	httpClient := httputil.NewClient(30, cfg.Director.SkipCertVerification)
	graphQLClient := gcli.NewClient(cfg.Director.URL, gcli.WithHTTPClient(httpClient))
	oauthClient := oauth.NewOauthClient(httpClient, cli, cfg.Director.OauthCredentialsSecretName, cfg.Director.Namespace)
	fatalOnError(oauthClient.WaitForCredentials())
	directorClient := director.NewDirectorClient(oauthClient, graphQLClient)

	// create storage
	var db storage.BrokerStorage
	if cfg.DbInMemory {
		db = storage.NewMemoryStorage()
	} else {
		store, conn, err := storage.NewFromConfig(cfg.Database, logs.WithField("service", "storage"))
		fatalOnError(err)
		db = store
		dbStatsCollector := sqlstats.NewStatsCollector("broker", conn)
		prometheus.MustRegister(dbStatsCollector)
	}

	// LMS
	fatalOnError(cfg.LMS.Validate())
	lmsClient := lms.NewClient(cfg.LMS, logs.WithField("service", "lmsClient"))
	lmsTenantManager := lms.NewTenantManager(db.LMSTenants(), lmsClient, logs.WithField("service", "lmsTenantManager"))

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

	runtimeProvider := runtime.NewComponentsListProvider(cfg.ManagedRuntimeComponentsYAMLFilePath)

	gardenerClusterConfig, err := gardener.NewGardenerClusterConfig(cfg.Gardener.KubeconfigPath)
	fatalOnError(err)
	gardenerSecrets, err := gardener.NewGardenerSecretsInterface(gardenerClusterConfig, cfg.Gardener.Project)
	fatalOnError(err)
	gardenerShoots, err := gardener.NewGardenerShootInterface(gardenerClusterConfig, cfg.Gardener.Project)
	fatalOnError(err)

	gardenerAccountPool := hyperscaler.NewAccountPool(gardenerSecrets)
	gardenerSharedPool := hyperscaler.NewSharedGardenerAccountPool(gardenerSecrets, gardenerShoots)
	accountProvider := hyperscaler.NewAccountProvider(nil, gardenerAccountPool, gardenerSharedPool)

	regions, err := provider.ReadPlatformRegionMappingFromFile(cfg.TrialRegionMappingFilePath)
	fatalOnError(err)
	logs.Infof("Platform region mapping for trial: %v", regions)
	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, runtimeProvider, cfg.Provisioning, cfg.KymaVersion, regions)
	fatalOnError(err)

	edpClient := edp.NewClient(cfg.EDP, logs.WithField("service", "edpClient"))

	avsClient, err := avs.NewClient(ctx, cfg.Avs, logs)
	fatalOnError(err)
	avsDel := avs.NewDelegator(avsClient, cfg.Avs, db.Operations())
	externalEvalAssistant := avs.NewExternalEvalAssistant(cfg.Avs)
	internalEvalAssistant := avs.NewInternalEvalAssistant(cfg.Avs)
	externalEvalCreator := provisioning.NewExternalEvalCreator(avsDel, cfg.Avs.Disabled, externalEvalAssistant)

	clientHTTPForIAS := httpClient
	if cfg.IAS.TLSRenegotiationEnable {
		clientHTTPForIAS = httputil.NewRenegotiationTLSClient(30, cfg.Director.SkipCertVerification)
	}
	bundleBuilder := ias.NewBundleBuilder(clientHTTPForIAS, cfg.IAS)
	iasTypeSetter := provisioning.NewIASType(bundleBuilder, cfg.IAS.Disabled)

	// application event broker
	eventBroker := event.NewPubSub()

	// metrics collectors
	metrics.RegisterAll(eventBroker, db.Operations(), db.Instances())

	// setup operation managers
	provisionManager := provisioning.NewManager(db.Operations(), eventBroker, logs.WithField("provisioning", "manager"))
	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))
	upgradeKymaManager := upgrade_kyma.NewManager(db.Operations(), eventBroker, logs.WithField("upgradeKyma", "manager"))

	// define steps
	kymaVersionConfigurator := provisioning.NewKymaVersionConfigurator(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	provisioningInit := provisioning.NewInitialisationStep(db.Operations(), db.Instances(),
		provisionerClient, directorClient, inputFactory, externalEvalCreator, iasTypeSetter, cfg.Provisioning.Timeout,
		kymaVersionConfigurator)
	provisionManager.InitStep(provisioningInit)

	provisioningSteps := []struct {
		disabled bool
		weight   int
		step     provisioning.Step
	}{
		{
			weight: 1,
			step:   provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider),
		},
		{
			weight:   1,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
		{
			weight: 1,
			step: provisioning.NewSkipForTrialPlanStep(db.Operations(),
				provisioning.NewProvideLmsTenantStep(lmsTenantManager, db.Operations(), cfg.LMS.Region, cfg.LMS.Mandatory)),
			disabled: cfg.LMS.Disabled,
		},
		{
			weight:   1,
			step:     provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			weight: 2,
			step: provisioning.NewSkipForTrialPlanStep(db.Operations(),
				provisioning.NewProvisionAzureEventHubStep(db.Operations(), azure.NewAzureProvider(), accountProvider, ctx)),
		},
		{
			weight: 2,
			step: provisioning.NewEnableForTrialPlanStep(db.Operations(),
				provisioning.NewNatsStreamingOverridesStep(db.Operations())),
		},
		{
			weight: 2,
			step:   provisioning.NewOverridesFromSecretsAndConfigStep(ctx, cli, db.Operations()),
		},
		{
			weight: 2,
			step:   provisioning.NewServiceManagerOverridesStep(db.Operations(), cfg.ServiceManager),
		},
		{
			weight: 2,
			step:   provisioning.NewAuditLogOverridesStep(db.Operations(), cfg.AuditLog),
		},
		{
			weight: 4,
			step: provisioning.NewSkipForTrialPlanStep(db.Operations(),
				provisioning.NewLmsCertificatesStep(lmsClient, db.Operations(), cfg.LMS.Mandatory)),
			disabled: cfg.LMS.Disabled,
		},
		{
			weight:   5,
			step:     provisioning.NewIASRegistrationStep(db.Operations(), bundleBuilder),
			disabled: cfg.IAS.Disabled,
		},
		{
			weight: 10,
			step:   provisioning.NewCreateRuntimeStep(db.Operations(), db.Instances(), provisionerClient),
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			provisionManager.AddStep(step.weight, step.step)
		}
	}

	deprovisioningInit := deprovisioning.NewInitialisationStep(db.Operations(), db.Instances(), provisionerClient)
	deprovisionManager.InitStep(deprovisioningInit)
	deprovisioningSteps := []struct {
		disabled bool
		weight   int
		step     deprovisioning.Step
	}{
		{
			weight: 1,
			step:   deprovisioning.NewAvsEvaluationsRemovalStep(avsDel, db.Operations(), externalEvalAssistant, internalEvalAssistant),
		},
		{
			weight: 1,
			step: deprovisioning.NewSkipForTrialPlanStep(db.Operations(),
				deprovisioning.NewDeprovisionAzureEventHubStep(db.Operations(), azure.NewAzureProvider(), accountProvider, ctx)),
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
			weight: 10,
			step:   deprovisioning.NewRemoveRuntimeStep(db.Operations(), db.Instances(), provisionerClient),
		},
	}
	for _, step := range deprovisioningSteps {
		if !step.disabled {
			deprovisionManager.AddStep(step.weight, step.step)
		}
	}

	upgradeKymaInit := upgrade_kyma.NewInitialisationStep(db.Operations(), db.Instances(), provisionerClient, inputFactory)
	upgradeKymaManager.InitStep(upgradeKymaInit)
	upgradeKymaSteps := []struct {
		disabled bool
		weight   int
		step     upgrade_kyma.Step
	}{
		{
			weight: 2,
			step:   upgrade_kyma.NewOverridesFromSecretsAndConfigStep(ctx, cli, db.Operations()),
		},
		{
			weight: 10,
			step:   upgrade_kyma.NewUpgradeKymaStep(db.Operations(), provisionerClient),
		},
	}
	for _, step := range upgradeKymaSteps {
		if !step.disabled {
			upgradeKymaManager.AddStep(step.weight, step.step)
		}
	}

	// run queues
	const workersAmount = 5
	provisionQueue := process.NewQueue(provisionManager, logs)
	provisionQueue.Run(ctx.Done(), workersAmount)

	deprovisionQueue := process.NewQueue(deprovisionManager, logs)
	deprovisionQueue.Run(ctx.Done(), workersAmount)

	upgradeKymaQueue := process.NewQueue(upgradeKymaManager, logs)
	upgradeKymaQueue.Run(ctx.Done(), workersAmount)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(dbmodel.OperationTypeProvision, db.Operations(), provisionQueue, logs)
		fatalOnError(err)
		err = processOperationsInProgressByType(dbmodel.OperationTypeDeprovision, db.Operations(), deprovisionQueue, logs)
		fatalOnError(err)
	} else {
		logger.Info("Skipping processing operation in progress on start")
	}

	plansValidator, err := broker.NewPlansSchemaValidator()
	fatalOnError(err)

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		broker.NewServices(cfg.Broker, optComponentsSvc, logs),
		broker.NewProvision(cfg.Broker, db.Operations(), db.Instances(), provisionQueue, inputFactory, plansValidator, cfg.EnableOnDemandVersion, logs),
		broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		broker.NewUpdate(logs),
		broker.NewGetInstance(db.Instances(), logs),
		broker.NewLastOperation(db.Operations(), logs),
		broker.NewBind(logs),
		broker.NewUnbind(logs),
		broker.NewGetBinding(logs),
		broker.NewLastBindingOperation(logs),
	}

	// create server
	router := mux.NewRouter()

	// create info endpoints
	respWriter := httputil.NewResponseWriter(logs, cfg.DevelopmentMode)
	runtimesInfoHandler := appinfo.NewRuntimeInfoHandler(db.Instances(), cfg.DefaultRequestRegion, respWriter)
	router.Handle("/info/runtimes", runtimesInfoHandler)

	// create metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	//gardenerClient, err := gardener.NewClient(gardenerClusterConfig)
	//fatalOnError(err)
	//runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient, "default", db.Instances(), logs)
	// TODO(upgrade): uncomment and inject upgradeKymaManager populated with steps
	//upgradeKymaManager := kyma.NewUpgradeKymaManager(db.Orchestrations(), nil, runtimeResolver, logs)
	//kymaQueue := process.NewQueue(upgradeKymaManager, logs)
	//kymaQueue.Run(ctx.Done(), workersAmount)
	//orchestrationHandler := orchestrate.NewOrchestrationHandler(db.Orchestrations(), kymaQueue, logs)

	// create OSB API endpoints
	router.Use(middleware.AddRegionToContext(cfg.DefaultRequestRegion))
	for _, prefix := range []string{
		"/oauth/",          // oauth2 handled by Ory
		"/oauth/{region}/", // oauth2 handled by Ory with region
	} {
		route := router.PathPrefix(prefix).Subrouter()
		broker.AttachRoutes(route, kymaEnvBroker, logger)
	}
	// TODO(upgrade): uncomment
	//orchestrationHandler.AttachRoutes(router)
	svr := handlers.CustomLoggingHandler(os.Stdout, router, func(writer io.Writer, params handlers.LogFormatterParams) {
		logs.Infof("Call handled: method=%s url=%s statusCode=%d size=%d", params.Request.Method, params.URL.Path, params.StatusCode, params.Size)
	})

	fatalOnError(http.ListenAndServe(cfg.Host+":"+cfg.Port, svr))
}

// queues all in progress provision operations existing in the database
func processOperationsInProgressByType(opType dbmodel.OperationType, op storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	operations, err := op.GetOperationsInProgressByType(opType)
	if err != nil {
		return errors.Wrap(err, "while getting in progress operations from storage")
	}
	for _, operation := range operations {
		queue.Add(operation.ID)
		log.Infof("Resuming the processing of %s operation ID: %s", opType, operation.ID)
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
