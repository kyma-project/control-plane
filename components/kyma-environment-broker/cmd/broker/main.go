package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	uaa "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/xsuaa"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"code.cloudfoundry.org/lager"
	"github.com/dlmiddlecote/sqlstats"
	gardenerclient "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/azure"
	orchestrationExt "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/appinfo"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/health"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/metrics"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
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

	// OperationTimeout is used to check on a top-level if any operation didn't exceed the time for processing.
	// It is used for provisioning and deprovisioning operations.
	OperationTimeout time.Duration `envconfig:"default=24h"`

	Host       string `envconfig:"optional"`
	Port       string `envconfig:"default=8080"`
	StatusPort string `envconfig:"default=8071"`

	Provisioning input.Config
	Director     director.Config
	Database     storage.Config
	Gardener     gardener.Config

	ServiceManager servicemanager.Config

	KymaVersion                          string
	EnableOnDemandVersion                bool `envconfig:"default=false"`
	ManagedRuntimeComponentsYAMLFilePath string
	DefaultRequestRegion                 string `envconfig:"default=cf-eu10"`

	Broker broker.Config

	Avs avs.Config
	LMS lms.Config
	IAS ias.Config
	EDP edp.Config

	// Service Manager services
	XSUAA struct {
		Disabled bool `envconfig:"default=true"`
	}
	Ems struct {
		Disabled bool `envconfig:"default=true"`
	}

	AuditLog auditlog.Config

	VersionConfig struct {
		Namespace string
		Name      string
	}

	TrialRegionMappingFilePath string
	MaxPaginationPage          int `envconfig:"default=100"`
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

	// create director client
	directorClient := director.NewDirectorClient(ctx, cfg.Director, logs.WithField("service", "directorClient"))

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
	gardenerClient, err := gardener.NewClient(gardenerClusterConfig)
	fatalOnError(err)
	gardenerSecrets, err := gardener.NewGardenerSecretsInterface(gardenerClusterConfig, cfg.Gardener.Project)
	fatalOnError(err)
	gardenerShoots, err := gardener.NewGardenerShootInterface(gardenerClusterConfig, cfg.Gardener.Project)
	fatalOnError(err)

	gardenerAccountPool := hyperscaler.NewAccountPool(gardenerSecrets, gardenerShoots)
	gardenerSharedPool := hyperscaler.NewSharedGardenerAccountPool(gardenerSecrets, gardenerShoots)
	accountProvider := hyperscaler.NewAccountProvider(gardenerAccountPool, gardenerSharedPool)

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
	internalEvalUpdater := provisioning.NewInternalEvalUpdater(avsDel, internalEvalAssistant, cfg.Avs)
	upgradeEvalManager := upgrade_kyma.NewEvaluationManager(avsDel, cfg.Avs)

	clientHTTPForIAS := httputil.NewClient(60, cfg.IAS.SkipCertVerification)
	if cfg.IAS.TLSRenegotiationEnable {
		clientHTTPForIAS = httputil.NewRenegotiationTLSClient(30, cfg.IAS.SkipCertVerification)
	}
	bundleBuilder := ias.NewBundleBuilder(clientHTTPForIAS, cfg.IAS)
	iasTypeSetter := provisioning.NewIASType(bundleBuilder, cfg.IAS.Disabled)

	// application event broker
	eventBroker := event.NewPubSub(logs)

	// metrics collectors
	metrics.RegisterAll(eventBroker, db.Operations(), db.Instances())

	//setup runtime overrides appender
	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)

	// setup operation managers
	provisionManager := provisioning.NewManager(db.Operations(), eventBroker, logs.WithField("provisioning", "manager"))
	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))

	serviceManagerClientFactory := servicemanager.NewClientFactory(cfg.ServiceManager)

	// define steps
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, accountVersionMapping)
	provisioningInit := provisioning.NewInitialisationStep(db.Operations(), db.Instances(),
		provisionerClient, directorClient, inputFactory, externalEvalCreator, internalEvalUpdater, iasTypeSetter,
		cfg.Provisioning.Timeout, cfg.OperationTimeout, runtimeVerConfigurator, serviceManagerClientFactory)
	provisionManager.InitStep(provisioningInit)

	provisioningSteps := []struct {
		disabled bool
		weight   int
		step     provisioning.Step
	}{
		{
			weight: 1,
			step: provisioning.NewServiceManagerOfferingStep("XSUAA_Offering",
				"xsuaa", "application", func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
					return &op.XSUAA.Instance
				}, db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight: 1,
			step: provisioning.NewServiceManagerOfferingStep("EMS_Offering",
				provisioning.EmsOfferingName, provisioning.EmsPlanName, func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
					return &op.Ems.Instance
				}, db.Operations()),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight: 2,
			step:   provisioning.NewResolveCredentialsStep(db.Operations(), accountProvider),
		},
		{
			weight: 2,
			step: provisioning.NewXSUAAProvisioningStep(db.Operations(), uaa.Config{
				// todo: set correct values from env variables
				DeveloperGroup:      "devGroup",
				DeveloperRole:       "devRole",
				NamespaceAdminGroup: "nag",
				NamespaceAdminRole:  "nar",
			}),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   2,
			step:     provisioning.NewEmsProvisionStep(db.Operations()),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight:   2,
			step:     provisioning.NewInternalEvaluationStep(avsDel, internalEvalAssistant),
			disabled: cfg.Avs.Disabled,
		},
		{
			weight: 2,
			step:   provisioning.NewLmsActivationStep(cfg.LMS, provisioning.NewProvideLmsTenantStep(lmsTenantManager, db.Operations(), cfg.LMS.Region, cfg.LMS.Mandatory)),
		},
		{
			weight:   2,
			step:     provisioning.NewEDPRegistrationStep(db.Operations(), edpClient, cfg.EDP),
			disabled: cfg.EDP.Disabled,
		},
		{
			weight: 3,
			step:   provisioning.NewSkipForTrialPlanStep(provisioning.NewProvisionAzureEventHubStep(db.Operations(), azure.NewAzureProvider(), accountProvider, ctx)),
		},
		{
			weight: 3,
			step:   provisioning.NewEnableForTrialPlanStep(provisioning.NewNatsStreamingOverridesStep()),
		},
		{
			weight: 3,
			step:   provisioning.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
		},
		{
			weight: 3,
			step:   provisioning.NewServiceManagerOverridesStep(db.Operations()),
		},
		{
			weight: 3,
			step:   provisioning.NewAuditLogOverridesStep(db.Operations(), cfg.AuditLog),
		},
		{
			weight: 5,
			step:   provisioning.NewLmsActivationStep(cfg.LMS, provisioning.NewLmsCertificatesStep(lmsClient, db.Operations(), cfg.LMS.Mandatory)),
		},
		{
			weight:   6,
			step:     provisioning.NewIASRegistrationStep(db.Operations(), bundleBuilder),
			disabled: cfg.IAS.Disabled,
		},
		{
			weight:   7,
			step:     provisioning.NewXSUAABindingStep(db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   7,
			step:     provisioning.NewEmsBindStep(db.Operations(), cfg.Database.SecretKey),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight: 10,
			step:   provisioning.NewCreateRuntimeStep(db.Operations(), db.RuntimeStates(), db.Instances(), provisionerClient),
		},
	}
	for _, step := range provisioningSteps {
		if !step.disabled {
			provisionManager.AddStep(step.weight, step.step)
		}
	}

	deprovisioningInit := deprovisioning.NewInitialisationStep(db.Operations(), db.Instances(), provisionerClient, accountProvider, serviceManagerClientFactory, cfg.OperationTimeout)
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
			step:   deprovisioning.NewSkipForTrialPlanStep(deprovisioning.NewDeprovisionAzureEventHubStep(db.Operations(), azure.NewAzureProvider(), accountProvider, ctx)),
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
			weight:   1,
			step:     deprovisioning.NewXSUAAUnbindStep(db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   1,
			step:     deprovisioning.NewEmsUnbindStep(db.Operations()),
			disabled: cfg.Ems.Disabled,
		},
		{
			weight:   2,
			step:     deprovisioning.NewXSUAADeprovisionStep(db.Operations()),
			disabled: cfg.XSUAA.Disabled,
		},
		{
			weight:   2,
			step:     deprovisioning.NewEmsDeprovisionStep(db.Operations()),
			disabled: cfg.Ems.Disabled,
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

	// run queues
	const workersAmount = 5
	provisionQueue := process.NewQueue(provisionManager, logs)
	provisionQueue.Run(ctx.Done(), workersAmount)

	deprovisionQueue := process.NewQueue(deprovisionManager, logs)
	deprovisionQueue.Run(ctx.Done(), workersAmount)

	plansValidator, err := broker.NewPlansSchemaValidator()
	fatalOnError(err)

	// create KymaEnvironmentBroker endpoints
	kymaEnvBroker := &broker.KymaEnvironmentBroker{
		broker.NewServices(cfg.Broker, optComponentsSvc, logs),
		broker.NewProvision(cfg.Broker, cfg.Gardener, db.Operations(), db.Instances(), provisionQueue, inputFactory, plansValidator, cfg.EnableOnDemandVersion, logs),
		broker.NewDeprovision(db.Instances(), db.Operations(), deprovisionQueue, logs),
		broker.NewUpdate(db.Instances(), logs),
		broker.NewGetInstance(db.Instances(), logs),
		broker.NewLastOperation(db.Operations(), db.Instances(), logs),
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

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.Gardener.Project)
	kymaQueue, err := NewOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, gardenerClient,
		gardenerNamespace, eventBroker, inputFactory, nil, time.Minute, runtimeVerConfigurator, cfg.DefaultRequestRegion, upgradeEvalManager, logs)
	fatalOnError(err)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, cfg.MaxPaginationPage, logs)

	if !cfg.DisableProcessOperationsInProgress {
		err = processOperationsInProgressByType(dbmodel.OperationTypeProvision, db.Operations(), provisionQueue, logs)
		fatalOnError(err)
		err = processOperationsInProgressByType(dbmodel.OperationTypeDeprovision, db.Operations(), deprovisionQueue, logs)
		fatalOnError(err)
		err = reprocessOrchestrations(db.Orchestrations(), db.Operations(), kymaQueue, logs)
		fatalOnError(err)
	} else {
		logger.Info("Skipping processing operation in progress on start")
	}

	// create OSB API endpoints
	router.Use(middleware.AddRegionToContext(cfg.DefaultRequestRegion))
	for _, prefix := range []string{
		"/oauth/",          // oauth2 handled by Ory
		"/oauth/{region}/", // oauth2 handled by Ory with region
	} {
		route := router.PathPrefix(prefix).Subrouter()
		broker.AttachRoutes(route, kymaEnvBroker, logger)
	}

	// create /orchestration
	orchestrationHandler.AttachRoutes(router)

	// create list runtimes endpoint
	runtimeHandler := runtime.NewHandler(db.Instances(), db.Operations(), cfg.MaxPaginationPage, cfg.DefaultRequestRegion)
	runtimeHandler.AttachRoutes(router)

	router.StrictSlash(true).PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("/swagger"))))
	svr := handlers.CustomLoggingHandler(os.Stdout, router, func(writer io.Writer, params handlers.LogFormatterParams) {
		logs.Infof("Call handled: method=%s url=%s statusCode=%d size=%d", params.Request.Method, params.URL.Path, params.StatusCode, params.Size)
	})

	fatalOnError(http.ListenAndServe(cfg.Host+":"+cfg.Port, svr))
}

// queues all in progress operations by type
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

func reprocessOrchestrations(orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	if err := processCancelingOrchestrations(orchestrationsStorage, operationsStorage, queue, log); err != nil {
		return errors.Wrap(err, "while processing canceled orchestrations")
	}
	if err := processOrchestration(orchestrationExt.InProgress, orchestrationsStorage, queue, log); err != nil {
		return errors.Wrap(err, "while processing in progress orchestrations")
	}
	if err := processOrchestration(orchestrationExt.Pending, orchestrationsStorage, queue, log); err != nil {
		return errors.Wrap(err, "while processing pending orchestrations")
	}
	return nil
}

func processOrchestration(state string, orchestrationsStorage storage.Orchestrations, queue *process.Queue, log logrus.FieldLogger) error {
	orchestrations, err := orchestrationsStorage.ListByState(state)
	if err != nil {
		return errors.Wrapf(err, "while getting %s orchestrations from storage", state)
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		queue.Add(o.OrchestrationID)
		log.Infof("Resuming the processing of %s orchestration ID: %s", state, o.OrchestrationID)
	}
	return nil
}

// processCancelingOrchestrations reprocess orchestrations with canceling state only when some in progress operations exists
// reprocess only one orchestration to not clog up the orchestration queue on start
func processCancelingOrchestrations(orchestrationsStorage storage.Orchestrations, operationsStorage storage.Operations, queue *process.Queue, log logrus.FieldLogger) error {
	orchestrations, err := orchestrationsStorage.ListByState(orchestrationExt.Canceling)
	if err != nil {
		return errors.Wrap(err, "while getting canceling orchestrations from storage")
	}
	sort.Slice(orchestrations, func(i, j int) bool {
		return orchestrations[i].CreatedAt.Before(orchestrations[j].CreatedAt)
	})

	for _, o := range orchestrations {
		ops, _, _, err := operationsStorage.ListUpgradeKymaOperationsByOrchestrationID(o.OrchestrationID, dbmodel.OperationFilter{States: []string{orchestrationExt.InProgress}})
		if err != nil {
			return errors.Wrapf(err, "while listing upgrade kyma operations for orchestration %s", o.OrchestrationID)
		}
		if len(ops) > 0 {
			log.Infof("Resuming the processing of %s orchestration ID: %s", orchestrationExt.Canceling, o.OrchestrationID)
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

func NewOrchestrationProcessingQueue(ctx context.Context, db storage.BrokerStorage,
	runtimeOverrides upgrade_kyma.RuntimeOverridesAppender, provisionerClient provisioner.Client,
	gardenerClient gardenerclient.CoreV1beta1Interface, gardenerNamespace string, pub event.Publisher,
	inputFactory input.CreatorForPlan, icfg *upgrade_kyma.TimeSchedule,
	pollingInterval time.Duration, runtimeVerConfigurator *runtimeversion.RuntimeVersionConfigurator,
	defaultRegion string, updateEvalManager *upgrade_kyma.EvaluationManager, logs logrus.FieldLogger) (*process.Queue, error) {

	upgradeKymaManager := upgrade_kyma.NewManager(db.Operations(), pub, logs.WithField("upgradeKyma", "manager"))
	upgradeKymaInit := upgrade_kyma.NewInitialisationStep(db.Operations(), db.Orchestrations(), db.Instances(),
		provisionerClient, inputFactory, updateEvalManager, icfg, runtimeVerConfigurator)

	upgradeKymaManager.InitStep(upgradeKymaInit)
	upgradeKymaSteps := []struct {
		disabled bool
		weight   int
		step     upgrade_kyma.Step
	}{
		{
			weight: 2,
			step:   upgrade_kyma.NewOverridesFromSecretsAndConfigStep(db.Operations(), runtimeOverrides, runtimeVerConfigurator),
		},
		{
			weight: 10,
			step:   upgrade_kyma.NewUpgradeKymaStep(db.Operations(), db.RuntimeStates(), provisionerClient, icfg),
		},
	}
	for _, step := range upgradeKymaSteps {
		if !step.disabled {
			upgradeKymaManager.AddStep(step.weight, step.step)
		}
	}

	runtimeLister := orchestration.NewRuntimeLister(db.Instances(), db.Operations(), runtime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestrationExt.NewGardenerRuntimeResolver(gardenerClient, gardenerNamespace, runtimeLister, logs)

	orchestrateKymaManager := kyma.NewUpgradeKymaManager(db.Orchestrations(), db.Operations(), db.Instances(),
		upgradeKymaManager, runtimeResolver, pollingInterval, logs)
	queue := process.NewQueue(orchestrateKymaManager, logs)

	// only one orchestration can be processed at the same time
	queue.Run(ctx.Done(), 1)

	return queue, nil
}
