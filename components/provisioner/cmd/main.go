package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/avast/retry-go"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/provisioner/internal/api"
	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/healthz"
	"github.com/kyma-project/control-plane/components/provisioner/internal/metrics"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"
	provisioningStages "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/database"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const connStringFormat string = "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s"

type config struct {
	Address               string `envconfig:"default=127.0.0.1:3000"`
	APIEndpoint           string `envconfig:"default=/graphql"`
	PlaygroundAPIEndpoint string `envconfig:"default=/graphql"`

	Database struct {
		User        string `envconfig:"default=postgres"`
		Password    string `envconfig:"default=password"`
		Host        string `envconfig:"default=localhost"`
		Port        string `envconfig:"default=5432"`
		Name        string `envconfig:"default=provisioner"`
		SSLMode     string `envconfig:"default=disable"`
		SSLRootCert string `envconfig:"optional"`
		SecretKey   string `envconfig:"optional"`
	}

	ProvisioningTimeout   queue.ProvisioningTimeouts
	DeprovisioningTimeout queue.DeprovisioningTimeouts
	HibernationTimeout    queue.HibernationTimeouts

	OperatorRoleBinding provisioningStages.OperatorRoleBinding

	Gardener struct {
		Project                                    string `envconfig:"default=gardenerProject"`
		KubeconfigPath                             string `envconfig:"default=./dev/kubeconfig.yaml"`
		AuditLogsPolicyConfigMap                   string `envconfig:"optional"`
		AuditLogsTenantConfigPath                  string `envconfig:"optional"`
		MaintenanceWindowConfigPath                string `envconfig:"optional"`
		ClusterCleanupResourceSelector             string `envconfig:"default=https://service-manager."`
		DefaultEnableKubernetesVersionAutoUpdate   bool   `envconfig:"default=false"`
		DefaultEnableMachineImageVersionAutoUpdate bool   `envconfig:"default=false"`
		DefaultEnableIMDSv2                        bool   `envconfig:"default=false"`
		EnableDumpShootSpec                        bool   `envconfig:"default=false"`
	}

	EnqueueInProgressOperations bool `envconfig:"default=true"`

	MetricsAddress string `envconfig:"default=127.0.0.1:9000"`

	LogLevel string `envconfig:"default=info"`
}

func (c *config) String() string {
	return fmt.Sprintf("Address: %s, APIEndpoint: %s, "+
		"DatabaseUser: %s, DatabaseHost: %s, DatabasePort: %s, "+
		"DatabaseName: %s, DatabaseSSLMode: %s, "+
		"ProvisioningTimeoutClusterCreation: %s "+
		"ProvisioningTimeoutInstallation: %s, ProvisioningTimeoutUpgrade: %s, "+
		"DeprovisioningNoInstallTimeoutClusterDeletion: %s, DeprovisioningNoInstallTimeoutWaitingForClusterDeletion: %s "+
		"ShootUpgradeTimeout: %s, "+
		"OperatorRoleBindingCreatingForAdmin: %t "+
		"GardenerProject: %s, GardenerKubeconfigPath: %s, GardenerAuditLogsPolicyConfigMap: %s, AuditLogsTenantConfigPath: %s, DefaultEnableIMDSv2: %v "+
		"EnqueueInProgressOperations: %v "+
		"EnableDumpShootSpec: %v "+
		"LogLevel: %s",
		c.Address, c.APIEndpoint,
		c.Database.User, c.Database.Host, c.Database.Port,
		c.Database.Name, c.Database.SSLMode,
		c.ProvisioningTimeout.ClusterCreation.String(),
		c.ProvisioningTimeout.Installation.String(), c.ProvisioningTimeout.Upgrade.String(),
		c.DeprovisioningTimeout.ClusterDeletion.String(), c.DeprovisioningTimeout.WaitingForClusterDeletion.String(),
		c.ProvisioningTimeout.ShootUpgrade.String(),

		c.OperatorRoleBinding.CreatingForAdmin,
		c.Gardener.Project, c.Gardener.KubeconfigPath, c.Gardener.AuditLogsPolicyConfigMap, c.Gardener.AuditLogsTenantConfigPath, c.Gardener.DefaultEnableIMDSv2,

		c.EnqueueInProgressOperations,
		c.Gardener.EnableDumpShootSpec,
		c.LogLevel)
}

func main() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	cfg := config{}
	err := envconfig.InitWithPrefix(&cfg, "APP")
	exitOnError(err, "Failed to load application config")

	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Warnf("Invalid log level: '%s', defaulting to 'info'", cfg.LogLevel)
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)

	log.Infof("Starting Provisioner")
	log.Infof("Config: %s", cfg.String())

	connString := fmt.Sprintf(connStringFormat, cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode, cfg.Database.SSLRootCert)

	connection, err := database.InitializeDatabaseConnection(connString, databaseConnectionRetries)
	exitOnError(err, "Failed to initialize persistence")

	dbsFactory, err := dbsession.NewFactory(connection, cfg.Database.SecretKey)

	exitOnError(err, "Cannot create database session")

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.Gardener.Project)

	gardenerClusterConfig, err := newGardenerClusterConfig(cfg)
	exitOnError(err, "Failed to initialize Gardener cluster client")

	gardenerClientSet, err := gardener.NewClient(gardenerClusterConfig)
	exitOnError(err, "Failed to create Gardener cluster clientset")

	gardenerClient, err := client.New(gardenerClusterConfig, client.Options{})
	exitOnError(err, "unable to create gardener client")

	k8sCoreClientSet, err := kubernetes.NewForConfig(gardenerClusterConfig)
	exitOnError(err, "Failed to create Kubernetes clientset")

	secretsInterface := k8sCoreClientSet.CoreV1().Secrets(gardenerNamespace)

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	k8sClientProvider := k8s.NewK8sClientProvider()

	adminKubeconfigRequest := gardenerClient.SubResource("adminkubeconfig")
	kubeconfigProvider := gardener.NewKubeconfigProvider(shootClient, adminKubeconfigRequest, secretsInterface)

	provisioningQueue := queue.CreateProvisioningQueue(cfg.ProvisioningTimeout, dbsFactory, shootClient, cfg.OperatorRoleBinding, k8sClientProvider, kubeconfigProvider)
	shootUpgradeQueue := queue.CreateShootUpgradeQueue(cfg.ProvisioningTimeout, dbsFactory, shootClient, cfg.OperatorRoleBinding, k8sClientProvider, kubeconfigProvider)
	deprovisioningQueue := queue.CreateDeprovisioningQueue(cfg.DeprovisioningTimeout, dbsFactory, shootClient)

	provisioner := gardener.NewProvisioner(gardenerNamespace, shootClient, dbsFactory, cfg.Gardener.AuditLogsPolicyConfigMap, cfg.Gardener.MaintenanceWindowConfigPath, cfg.Gardener.EnableDumpShootSpec)
	shootController, err := newShootController(gardenerNamespace, gardenerClusterConfig, dbsFactory, cfg.Gardener.AuditLogsTenantConfigPath)
	exitOnError(err, "Failed to create Shoot controller.")
	go func() {
		err := shootController.StartShootController()
		exitOnError(err, "Failed to start Shoot Controller")
	}()

	provisioningSVC := newProvisioningService(
		cfg.Gardener.Project,
		provisioner,
		dbsFactory,
		gardener.NewShootProvider(shootClient),
		provisioningQueue,
		deprovisioningQueue,
		shootUpgradeQueue,
		cfg.Gardener.DefaultEnableKubernetesVersionAutoUpdate,
		cfg.Gardener.DefaultEnableMachineImageVersionAutoUpdate,
		cfg.Gardener.DefaultEnableIMDSv2,
		kubeconfigProvider,
	)

	tenantUpdater := api.NewTenantUpdater(dbsFactory.NewReadWriteSession())
	validator := api.NewValidator()
	resolver := api.NewResolver(provisioningSVC, validator, tenantUpdater)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provisioningQueue.Run(ctx.Done())

	deprovisioningQueue.Run(ctx.Done())

	shootUpgradeQueue.Run(ctx.Done())

	gqlCfg := gqlschema.Config{
		Resolvers: resolver,
	}
	executableSchema := gqlschema.NewExecutableSchema(gqlCfg)

	presenter := apperrors.NewPresenter(log.StandardLogger())

	log.Infof("Registering endpoint on %s...", cfg.APIEndpoint)
	router := mux.NewRouter()
	router.Use(middlewares.ExtractTenant)

	router.HandleFunc("/", playground.Handler("Dataloader", cfg.PlaygroundAPIEndpoint))

	gqlHandler := handler.New(executableSchema)
	gqlHandler.AddTransport(transport.POST{})
	gqlHandler.AddTransport(transport.GET{})
	gqlHandler.Use(extension.Introspection{})
	gqlHandler.SetErrorPresenter(presenter.Do)
	router.Handle(cfg.APIEndpoint, gqlHandler)
	router.HandleFunc("/healthz", healthz.NewHTTPHandler(log.StandardLogger()))

	// Metrics
	err = metrics.Register(dbsFactory.NewReadSession())
	exitOnError(err, "Failed to register metrics collectors")

	// Expose metrics on different port as it cannot be secured with mTLS
	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Handler: metricsRouter,
		Addr:    cfg.MetricsAddress,
	}

	log.Infof("API listening on %s...", cfg.Address)
	log.Infof("Metrics API listening on %s...", cfg.MetricsAddress)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		if err := http.ListenAndServe(cfg.Address, router); err != nil {
			log.Errorf("Error starting server: %s", err.Error())
		}
	}()

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			log.Errorf("Error starting metrics server: %s", err.Error())
		}
	}()

	if cfg.EnqueueInProgressOperations {
		err = enqueueOperationsInProgress(dbsFactory, provisioningQueue, deprovisioningQueue, shootUpgradeQueue)
		exitOnError(err, "Failed to enqueue in progress operations")
	}

	wg.Wait()
}

func enqueueOperationsInProgress(dbFactory dbsession.Factory, provisioningQueue, deprovisioningQueue, shootUpgradeQueue queue.OperationQueue) error {
	readSession := dbFactory.NewReadSession()

	var inProgressOps []model.Operation
	var err error

	// Due to Schema Migrator running post upgrade the pod will be in crash loop back off and Helm deployment will not finish
	// therefor we need to wait for schema to be initialized in case of blank installation.
	err = retry.Do(func() error {
		inProgressOps, err = readSession.ListInProgressOperations()
		if err != nil {
			log.Warnf("failed to list in progress operation")
			return err
		}
		return nil
	}, retry.Attempts(30), retry.DelayType(retry.FixedDelay), retry.Delay(5*time.Second))
	if err != nil {
		return fmt.Errorf("error enqueuing in progress operations: %s", err.Error())
	}

	for _, op := range inProgressOps {
		switch op.Type {
		case model.ProvisionNoInstall:
			provisioningQueue.Add(op.ID)
		case model.DeprovisionNoInstall:
			deprovisioningQueue.Add(op.ID)
		case model.UpgradeShoot:
			shootUpgradeQueue.Add(op.ID)
		}
	}

	return nil
}

func exitOnError(err error, context string) {
	if err != nil {
		wrappedError := errors.Wrap(err, context)
		log.Fatal(wrappedError)
	}
}
