package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/avast/retry-go"
	"github.com/gorilla/mux"
	installationSDK "github.com/kyma-incubator/hydroform/install/installation"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api"
	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/healthz"
	"github.com/kyma-project/control-plane/components/provisioner/internal/installation"
	"github.com/kyma-project/control-plane/components/provisioner/internal/metrics"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"
	provisioningStages "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/persistence/database"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"
	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util/k8s"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const connStringFormat string = "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s"

type config struct {
	Address                      string `envconfig:"default=127.0.0.1:3000"`
	APIEndpoint                  string `envconfig:"default=/graphql"`
	PlaygroundAPIEndpoint        string `envconfig:"default=/graphql"`
	DirectorURL                  string `envconfig:"default=http://compass-director.compass-system.svc.cluster.local:3000/graphql"`
	SkipDirectorCertVerification bool   `envconfig:"default=false"`
	DirectorOAuthPath            string `envconfig:"APP_DIRECTOR_OAUTH_PATH,default=./dev/director.yaml"`

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
	}

	LatestDownloadedReleases int  `envconfig:"default=5"`
	DownloadPreReleases      bool `envconfig:"default=true"`

	EnqueueInProgressOperations bool `envconfig:"default=true"`

	MetricsAddress string `envconfig:"default=127.0.0.1:9000"`

	LogLevel string `envconfig:"default=info"`

	// TODO: Remove after data migration
	RunAwsConfigMigration bool `envconfig:"default=false"`
}

func (c *config) String() string {
	return fmt.Sprintf("Address: %s, APIEndpoint: %s, DirectorURL: %s, "+
		"SkipDirectorCertVerification: %v, DirectorOAuthPath: %s, "+
		"DatabaseUser: %s, DatabaseHost: %s, DatabasePort: %s, "+
		"DatabaseName: %s, DatabaseSSLMode: %s, "+
		"ProvisioningTimeoutClusterCreation: %s "+
		"ProvisioningTimeoutInstallation: %s, ProvisioningTimeoutUpgrade: %s, "+
		"ProvisioningTimeoutAgentConfiguration: %s, ProvisioningTimeoutAgentConnection: %s, "+
		"DeprovisioningNoInstallTimeoutClusterDeletion: %s, DeprovisioningNoInstallTimeoutWaitingForClusterDeletion: %s "+
		"ShootUpgradeTimeout: %s, "+
		"OperatorRoleBindingL2SubjectName: %s, OperatorRoleBindingL3SubjectName: %s, OperatorRoleBindingCreatingForAdmin: %t "+
		"GardenerProject: %s, GardenerKubeconfigPath: %s, GardenerAuditLogsPolicyConfigMap: %s, AuditLogsTenantConfigPath: %s, "+
		"LatestDownloadedReleases: %d, DownloadPreReleases: %v, "+
		"EnqueueInProgressOperations: %v"+
		"LogLevel: %s"+
		"RunAwsConfigMigration: %v",
		c.Address, c.APIEndpoint, c.DirectorURL,
		c.SkipDirectorCertVerification, c.DirectorOAuthPath,
		c.Database.User, c.Database.Host, c.Database.Port,
		c.Database.Name, c.Database.SSLMode,
		c.ProvisioningTimeout.ClusterCreation.String(),
		c.ProvisioningTimeout.Installation.String(), c.ProvisioningTimeout.Upgrade.String(),
		c.ProvisioningTimeout.AgentConfiguration.String(), c.ProvisioningTimeout.AgentConnection.String(),
		c.DeprovisioningTimeout.ClusterDeletion.String(), c.DeprovisioningTimeout.WaitingForClusterDeletion.String(),
		c.ProvisioningTimeout.ShootUpgrade.String(),
		c.OperatorRoleBinding.L2SubjectName, c.OperatorRoleBinding.L3SubjectName, c.OperatorRoleBinding.CreatingForAdmin,
		c.Gardener.Project, c.Gardener.KubeconfigPath, c.Gardener.AuditLogsPolicyConfigMap, c.Gardener.AuditLogsTenantConfigPath,
		c.LatestDownloadedReleases, c.DownloadPreReleases,
		c.EnqueueInProgressOperations,
		c.LogLevel, c.RunAwsConfigMigration)
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

	k8sCoreClientSet, err := kubernetes.NewForConfig(gardenerClusterConfig)
	exitOnError(err, "Failed to create Kubernetes clientset")

	secretsInterface := k8sCoreClientSet.CoreV1().Secrets(gardenerNamespace)

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	installationHandlerConstructor := func(c *rest.Config, o ...installationSDK.InstallationOption) (installationSDK.Installer, error) {
		return installationSDK.NewKymaInstaller(c, o...)
	}

	installationService := installation.NewInstallationService(cfg.ProvisioningTimeout.Installation, installationHandlerConstructor, cfg.Gardener.ClusterCleanupResourceSelector)

	directorClient, err := newDirectorClient(cfg)
	exitOnError(err, "Failed to initialize Director client")

	k8sClientProvider := k8s.NewK8sClientProvider()

	runtimeConfigurator := runtime.NewRuntimeConfigurator(k8sClientProvider, directorClient)

	provisioningQueue := queue.CreateProvisioningQueue(
		cfg.ProvisioningTimeout,
		dbsFactory,
		directorClient,
		shootClient,
		secretsInterface,
		cfg.OperatorRoleBinding,
		k8sClientProvider,
		runtimeConfigurator)

	upgradeQueue := queue.CreateUpgradeQueue(cfg.ProvisioningTimeout, dbsFactory, directorClient, installationService)

	deprovisioningQueue := queue.CreateDeprovisioningQueue(cfg.DeprovisioningTimeout, dbsFactory, directorClient, shootClient)

	shootUpgradeQueue := queue.CreateShootUpgradeQueue(cfg.ProvisioningTimeout, dbsFactory, directorClient, shootClient, cfg.OperatorRoleBinding, k8sClientProvider, secretsInterface)

	hibernationQueue := queue.CreateHibernationQueue(cfg.HibernationTimeout, dbsFactory, directorClient, shootClient)

	provisioner := gardener.NewProvisioner(gardenerNamespace, shootClient, dbsFactory, cfg.Gardener.AuditLogsPolicyConfigMap, cfg.Gardener.MaintenanceWindowConfigPath)
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
		directorClient,
		installationService,
		gardener.NewShootProvider(shootClient),
		provisioningQueue,
		deprovisioningQueue,
		upgradeQueue,
		shootUpgradeQueue,
		hibernationQueue,
		cfg.Gardener.DefaultEnableKubernetesVersionAutoUpdate,
		cfg.Gardener.DefaultEnableMachineImageVersionAutoUpdate)

	tenantUpdater := api.NewTenantUpdater(dbsFactory.NewReadWriteSession())
	validator := api.NewValidator()
	resolver := api.NewResolver(provisioningSVC, validator, tenantUpdater)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provisioningQueue.Run(ctx.Done())

	deprovisioningQueue.Run(ctx.Done())

	upgradeQueue.Run(ctx.Done())

	shootUpgradeQueue.Run(ctx.Done())

	hibernationQueue.Run(ctx.Done())

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
		err = enqueueOperationsInProgress(dbsFactory, provisioningQueue, deprovisioningQueue, upgradeQueue, shootUpgradeQueue, hibernationQueue)
		exitOnError(err, "Failed to enqueue in progress operations")
	}

	wg.Wait()
}

func enqueueOperationsInProgress(dbFactory dbsession.Factory, provisioningQueue, deprovisioningQueue, upgradeQueue, shootUpgradeQueue, hibernationQueue queue.OperationQueue) error {
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
		case model.Upgrade:
			upgradeQueue.Add(op.ID)
		case model.Hibernate:
			hibernationQueue.Add(op.ID)
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
