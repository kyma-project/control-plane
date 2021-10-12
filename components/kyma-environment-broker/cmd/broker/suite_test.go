package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/Peripli/service-manager-cli/pkg/types"
	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerFake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	hyperscalerautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/auditlog"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring"
	monitoringmocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring/mocks"
	kebOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	kebRuntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	globalAccountLabel     = "account"
	subAccountLabel        = "subaccount"
	runtimeIDAnnotation    = "kcp.provisioner.kyma-project.io/runtime-id"
	defaultNamespace       = "kcp-system"
	defaultKymaVer         = "1.21"
	kymaVersionsConfigName = "kyma-versions"
	defaultRegion          = "cf-eu10"
	globalAccountID        = "dummy-ga-id"
	dashboardURL           = "http://console.garden-dummy.kyma.io"
	brokerID               = "fake-broker-id"
	operationID            = "provisioning-op-id"
	deprovisioningOpID     = "deprovisioning-op-id"
	instanceID             = "instance-id"
	smRegion               = "eu"
	dbSecretKey            = "1234567890123456"

	subscriptionNameRegular = "regular"
	subscriptionNameShared  = "shared"

	pollingInterval = 3 * time.Millisecond
)

type OrchestrationSuite struct {
	gardenerNamespace string
	provisionerClient *provisioner.FakeClient
	kymaQueue         *process.Queue
	clusterQueue      *process.Queue
	storage           storage.BrokerStorage
	gardenerClient    *gardenerFake.Clientset

	t *testing.T
}

func NewOrchestrationSuite(t *testing.T, additionalKymaVersions []string) *OrchestrationSuite {
	logs := logrus.New()
	logs.Formatter.(*logrus.TextFormatter).TimestampFormat = "15:04:05.000"

	var cfg Config
	cfg.Connectivity.Disabled = true
	cfg.AuditLog.Disabled = false
	cfg.AuditLog = auditlog.Config{
		URL:           "https://host1:8080/aaa/v2/",
		User:          "fooUser",
		Password:      "barPass",
		Tenant:        "fooTen",
		EnableSeqHttp: true,
	}

	//auditLog create file here.
	inMemoryFs, err := createInMemFS()
	require.NoError(t, err)

	//monitoring config and client
	cfg.Monitoring = monitoring.Config{
		Namespace:       "mornitoring",
		ChartUrl:        "notEmptyChart",
		RemoteWriteUrl:  "notEmptyUrl",
		RemoteWritePath: "notEmptyPath",
		Disabled:        false,
	}
	monitoringClient := &monitoringmocks.Client{}
	monitoringClient.On("IsPresent", mock.Anything).Return(true, nil)
	monitoringClient.On("UpgradeRelease", mock.Anything).Return(nil, nil)

	optionalComponentsDisablers := kebRuntime.ComponentsDisablers{}
	optComponentsSvc := kebRuntime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := kebRuntime.NewDisabledComponentsProvider()

	componentListProvider := &automock.ComponentListProvider{}
	componentListProvider.On("AllComponents", mock.Anything).Return([]v1alpha1.KymaComponent{}, nil)

	oidcDefaults := fixture.FixOIDCConfigDTO()

	kymaVer := "1.15.1"
	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentListProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		ProvisioningTimeout:         time.Minute,
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, kymaVer, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders, oidcDefaults, fixture.FixDNSConfigDTO())
	require.NoError(t, err)

	ctx, _ := context.WithTimeout(context.Background(), 20*time.Minute)
	db := storage.NewMemoryStorage()
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	cli := fake.NewFakeClientWithScheme(sch, fixK8sResources(kymaVer, additionalKymaVersions)...)

	reconcilerClient := reconciler.NewFakeClient()
	gardenerClient := gardenerFake.NewSimpleClientset()
	provisionerClient := provisioner.NewFakeClient()
	const gardenerProject = "testing"
	gardenerNamespace := fmt.Sprintf("garden-%s", gardenerProject)

	eventBroker := event.NewPubSub(logs)

	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)

	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(kymaVer, "", runtimeversion.NewAccountVersionMapping(ctx, cli, defaultNamespace, kymaVersionsConfigName, logs))

	avsClient, _ := avs.NewClient(ctx, avs.Config{}, logs)
	avsDel := avs.NewDelegator(avsClient, avs.Config{}, db.Operations())
	upgradeEvaluationManager := avs.NewEvaluationManager(avsDel, avs.Config{})
	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient.CoreV1beta1(), gardenerNamespace, runtimeLister, logs)

	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, &upgrade_kyma.TimeSchedule{
		Retry:              10 * time.Millisecond,
		StatusCheck:        100 * time.Millisecond,
		UpgradeKymaTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeVerConfigurator, runtimeResolver, upgradeEvaluationManager,
		&cfg, hyperscaler.NewAccountProvider(nil, nil), reconcilerClient, nil, inMemoryFs, monitoringClient, logs, cli)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 10 * time.Millisecond,
		StatusCheck:           100 * time.Millisecond,
		UpgradeClusterTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, upgradeEvaluationManager, logs, cli, cfg)

	kymaQueue.SpeedUp(1000)
	clusterQueue.SpeedUp(1000)

	return &OrchestrationSuite{
		gardenerNamespace: gardenerNamespace,
		provisionerClient: provisionerClient,
		kymaQueue:         kymaQueue,
		clusterQueue:      clusterQueue,
		storage:           db,
		gardenerClient:    gardenerClient,

		t: t,
	}
}

type RuntimeOptions struct {
	GlobalAccountID  string
	SubAccountID     string
	PlatformProvider internal.CloudProvider
	PlatformRegion   string
	Region           string
	PlanID           string
	ZonesCount       *int
	Provider         internal.CloudProvider
	KymaVersion      string
	OverridesVersion string
	OIDC             *internal.OIDCConfigDTO
	UserID           string
	RuntimeAdmins    []string
}

func (o *RuntimeOptions) ProvideGlobalAccountID() string {
	if o.GlobalAccountID != "" {
		return o.GlobalAccountID
	} else {
		return uuid.New().String()
	}
}

func (o *RuntimeOptions) ProvideSubAccountID() string {
	if o.SubAccountID != "" {
		return o.SubAccountID
	} else {
		return uuid.New().String()
	}
}

func (o *RuntimeOptions) ProvidePlatformRegion() string {
	if o.PlatformProvider != "" {
		return o.PlatformRegion
	} else {
		return "cf-eu10"
	}
}

func (o *RuntimeOptions) ProvideRegion() *string {
	if o.Region != "" {
		return &o.Region
	} else {
		r := "westeurope"
		return &r
	}
}

func (o *RuntimeOptions) ProvidePlanID() string {
	if o.PlanID == "" {
		return broker.AzurePlanID
	} else {
		return o.PlanID
	}
}

func (o *RuntimeOptions) ProvideZonesCount() *int {
	return o.ZonesCount
}

func (o *RuntimeOptions) ProvideOIDC() *internal.OIDCConfigDTO {
	if o.OIDC != nil {
		return o.OIDC
	} else {
		return nil
	}
}

func (o *RuntimeOptions) ProvideUserID() string {
	return o.UserID
}

func (o *RuntimeOptions) ProvideRuntimeAdmins() []string {
	if o.RuntimeAdmins != nil {
		return o.RuntimeAdmins
	} else {
		return nil
	}
}

func (s *OrchestrationSuite) CreateProvisionedRuntime(options RuntimeOptions) string {
	runtimeID := uuid.New().String()
	shootName := fmt.Sprintf("shoot%s", runtimeID)
	planID := options.ProvidePlanID()
	planName := broker.AzurePlanName
	globalAccountID := options.ProvideGlobalAccountID()
	subAccountID := options.ProvideSubAccountID()
	instanceID := uuid.New().String()
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: planID,
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
		},
		PlatformRegion: options.ProvidePlatformRegion(),
		Parameters: internal.ProvisioningParametersDTO{
			Region: options.ProvideRegion(),
		},
	}

	instance := internal.Instance{
		RuntimeID:       runtimeID,
		ServicePlanID:   planID,
		ServicePlanName: planName,
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    subAccountID,
		Parameters:      provisioningParameters,
		ProviderRegion:  options.ProvidePlatformRegion(),
		InstanceDetails: internal.InstanceDetails{
			RuntimeID:   runtimeID,
			ShootName:   shootName,
			ShootDomain: "fake.domain",
		},
	}

	provisioningOperation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			State:                  domain.Succeeded,
			ID:                     uuid.New().String(),
			InstanceID:             instanceID,
			ProvisioningParameters: provisioningParameters,
			InstanceDetails: internal.InstanceDetails{
				RuntimeID:   instance.RuntimeID,
				ShootName:   shootName,
				ShootDomain: "fake.domain",
			},
		},
	}
	shoot := &gardenerapi.Shoot{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      shootName,
			Namespace: s.gardenerNamespace,
			Labels: map[string]string{
				globalAccountLabel: globalAccountID,
				subAccountLabel:    subAccountID,
			},
			Annotations: map[string]string{
				runtimeIDAnnotation: runtimeID,
			},
		},
		Spec: gardenerapi.ShootSpec{
			Region: options.ProvidePlatformRegion(),
			Maintenance: &gardenerapi.Maintenance{
				TimeWindow: &gardenerapi.MaintenanceTimeWindow{
					Begin: "030000+0000",
					End:   "040000+0000",
				},
			},
		},
	}

	require.NoError(s.t, s.storage.Instances().Insert(instance))
	require.NoError(s.t, s.storage.Operations().InsertProvisioningOperation(provisioningOperation))
	_, err := s.gardenerClient.CoreV1beta1().Shoots(s.gardenerNamespace).Create(context.Background(), shoot, v1.CreateOptions{})
	require.NoError(s.t, err)
	return runtimeID
}

func (s *OrchestrationSuite) createOrchestration(oType orchestration.Type, queue *process.Queue, params orchestration.Parameters) string {
	now := time.Now()
	o := internal.Orchestration{
		OrchestrationID: uuid.New().String(),
		Type:            oType,
		State:           orchestration.Pending,
		Description:     "started processing of Kyma upgrade",
		Parameters:      params,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	require.NoError(s.t, s.storage.Orchestrations().Insert(o))

	queue.Add(o.OrchestrationID)
	return o.OrchestrationID
}

func (s *OrchestrationSuite) CreateUpgradeKymaOrchestration(params orchestration.Parameters) string {
	return s.createOrchestration(orchestration.UpgradeKymaOrchestration, s.kymaQueue, params)
}

func (s *OrchestrationSuite) CreateUpgradeClusterOrchestration(params orchestration.Parameters) string {
	return s.createOrchestration(orchestration.UpgradeClusterOrchestration, s.clusterQueue, params)
}

func (s *OrchestrationSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
	err := wait.Poll(time.Millisecond*100, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *OrchestrationSuite) FinishUpgradeOperationByProvisioner(runtimeID string) {
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgrade, runtimeID)
}

func (s *OrchestrationSuite) FinishUpgradeShootOperationByProvisioner(runtimeID string) {
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgradeShoot, runtimeID)
}

func (s *OrchestrationSuite) WaitForOrchestrationState(orchestrationID string, state string) {
	var orchestration *internal.Orchestration
	err := wait.PollImmediate(100*time.Millisecond, 2*time.Second, func() (done bool, err error) {
		orchestration, _ = s.storage.Orchestrations().GetByID(orchestrationID)
		return orchestration.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the orchestration expected state %s. The existing orchestration %+v", state, orchestration)
}

func (s *OrchestrationSuite) AssertRuntimeUpgraded(runtimeID string, version string) {
	assert.True(s.t, s.provisionerClient.IsRuntimeUpgraded(runtimeID, version), "The runtime %s expected to be upgraded", runtimeID)
}

func (s *OrchestrationSuite) AssertRuntimeNotUpgraded(runtimeID string) {
	assert.False(s.t, s.provisionerClient.IsRuntimeUpgraded(runtimeID, ""), "The runtime %s expected to be not upgraded", runtimeID)
}

func (s *OrchestrationSuite) AssertShootUpgraded(runtimeID string) {
	assert.True(s.t, s.provisionerClient.IsShootUpgraded(runtimeID), "The shoot %s expected to be upgraded", runtimeID)
}

func (s *OrchestrationSuite) AssertShootNotUpgraded(runtimeID string) {
	assert.False(s.t, s.provisionerClient.IsShootUpgraded(runtimeID), "The shoot %s expected to be not upgraded", runtimeID)
}

func fixK8sResources(defaultKymaVersion string, additionalKymaVersions []string) []runtime.Object {
	var resources []runtime.Object
	override := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "overrides",
			Namespace: "kcp-system",
			Labels: map[string]string{
				fmt.Sprintf("overrides-version-%s", defaultKymaVersion): "true",
				"overrides-plan-azure":    "true",
				"overrides-plan-trial":    "true",
				"overrides-plan-aws":      "true",
				"overrides-plan-free":     "true",
				"overrides-plan-azure_ha": "true",
				"overrides-plan-aws_ha":   "true",
				"overrides-plan-preview":  "true",
			},
		},
		Data: map[string]string{
			"foo": "bar",
		},
	}
	scOverride := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "service-catalog2-overrides",
			Namespace: "kcp-system",
			Labels: map[string]string{
				fmt.Sprintf("overrides-version-%s", defaultKymaVersion): "true",
				"overrides-plan-azure":    "true",
				"overrides-plan-trial":    "true",
				"overrides-plan-aws":      "true",
				"overrides-plan-free":     "true",
				"overrides-plan-azure_ha": "true",
				"overrides-plan-aws_ha":   "true",
				"overrides-plan-preview":  "true",
				"component":               "service-catalog2",
			},
		},
		Data: map[string]string{
			"setting-one": "1234",
		},
	}
	for _, version := range additionalKymaVersions {
		override.ObjectMeta.Labels[fmt.Sprintf("overrides-version-%s", version)] = "true"
		scOverride.ObjectMeta.Labels[fmt.Sprintf("overrides-version-%s", version)] = "true"
	}
	resources = append(resources, override, scOverride)

	return resources
}

type ProvisioningSuite struct {
	provisionerClient   *provisioner.FakeClient
	provisioningManager *provisioning.StagedManager
	provisioningQueue   *process.Queue
	storage             storage.BrokerStorage
	directorClient      *director.FakeClient

	t         *testing.T
	avsServer *avs.MockAvsServer
}

func NewProvisioningSuite(t *testing.T) *ProvisioningSuite {
	ctx, _ := context.WithTimeout(context.Background(), 20*time.Minute)
	logs := logrus.New()
	db := storage.NewMemoryStorage()

	cfg := fixConfig()

	//auditLog create file here.
	inMemoryFs, err := createInMemFS()
	require.NoError(t, err)

	provisionerClient := provisioner.NewFakeClient()

	optionalComponentsDisablers := kebRuntime.ComponentsDisablers{}
	optComponentsSvc := kebRuntime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := kebRuntime.NewDisabledComponentsProvider()

	componentListProvider := &automock.ComponentListProvider{}
	componentListProvider.On("AllComponents", mock.Anything).Return([]v1alpha1.KymaComponent{}, nil)

	oidcDefaults := fixture.FixOIDCConfigDTO()

	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentListProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		ProvisioningTimeout:         time.Minute,
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders, oidcDefaults, fixture.FixDNSConfigDTO())

	require.NoError(t, err)

	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	additionalKymaVersions := []string{"1.19", "1.20", "main"}
	cli := fake.NewFakeClientWithScheme(sch, fixK8sResources(defaultKymaVer, additionalKymaVersions)...)

	server := avs.NewMockAvsServer(t)
	mockServer := avs.FixMockAvsServer(server)
	avsConfig := avs.Config{
		OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
		ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
	}

	client, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	assert.NoError(t, err)
	avsDel := avs.NewDelegator(client, avsConfig, db.Operations())
	externalEvalAssistant := avs.NewExternalEvalAssistant(cfg.Avs)
	internalEvalAssistant := avs.NewInternalEvalAssistant(cfg.Avs)
	externalEvalCreator := provisioning.NewExternalEvalCreator(avsDel, cfg.Avs.Disabled, externalEvalAssistant)
	internalEvalUpdater := provisioning.NewInternalEvalUpdater(avsDel, internalEvalAssistant, cfg.Avs)

	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, "", accountVersionMapping)

	iasFakeClient := ias.NewFakeClient()
	bundleBuilder := ias.NewBundleBuilder(iasFakeClient, cfg.IAS)

	edpClient := edp.NewFakeClient()

	monitoringClient := &monitoringmocks.Client{}
	monitoringClient.On("IsDeployed", mock.Anything).Return(false, nil)
	monitoringClient.On("IsPresent", mock.Anything).Return(false, nil)
	monitoringClient.On("InstallRelease", mock.Anything).Return(nil, nil)

	accountProvider := fixAccountProvider()

	smcf := fixServiceManagerFactory()

	directorClient := director.NewFakeClient()

	reconcilerClient := reconciler.NewFakeClient()

	eventBroker := event.NewPubSub(logs)

	provisionManager := provisioning.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(ctx, provisionManager, workersAmount, cfg, db, provisionerClient,
		directorClient, inputFactory, avsDel, internalEvalAssistant, externalEvalCreator, internalEvalUpdater, runtimeVerConfigurator,
		runtimeOverrides, smcf, bundleBuilder, edpClient, monitoringClient, accountProvider, inMemoryFs, reconcilerClient, logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	return &ProvisioningSuite{
		provisionerClient:   provisionerClient,
		provisioningManager: provisionManager,
		provisioningQueue:   provisioningQueue,
		storage:             db,
		directorClient:      directorClient,
		avsServer:           server,

		t: t,
	}
}

func (s *ProvisioningSuite) CreateProvisioning(options RuntimeOptions) string {
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: options.ProvidePlanID(),
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    options.ProvideSubAccountID(),
			UserID:          options.ProvideUserID(),
			ServiceManager: &internal.ServiceManagerEntryDTO{
				URL: "sm_url",
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: "sm_username",
						Password: "sm_password",
					},
				},
			},
		},
		PlatformProvider: options.PlatformProvider,
		Parameters: internal.ProvisioningParametersDTO{
			Region:                options.ProvideRegion(),
			ZonesCount:            options.ProvideZonesCount(),
			KymaVersion:           options.KymaVersion,
			OverridesVersion:      options.OverridesVersion,
			OIDC:                  options.ProvideOIDC(),
			RuntimeAdministrators: options.ProvideRuntimeAdmins(),
		},
	}

	shootName := gardener.CreateShootName()

	operation, err := internal.NewProvisioningOperationWithID(operationID, instanceID, provisioningParameters)
	require.NoError(s.t, err)
	operation.ShootName = shootName
	operation.ShootDomain = fmt.Sprintf("%s.%s.%s", shootName, "garden-dummy", strings.Trim("kyma.io", "."))
	operation.DashboardURL = dashboardURL
	operation.State = orchestration.Pending

	err = s.storage.Operations().InsertProvisioningOperation(operation)
	require.NoError(s.t, err)

	err = s.storage.Instances().Insert(internal.Instance{
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    "dummy-sa",
		ServiceID:       provisioningParameters.ServiceID,
		ServiceName:     broker.KymaServiceName,
		ServicePlanID:   provisioningParameters.PlanID,
		ServicePlanName: broker.AzurePlanName,
		DashboardURL:    dashboardURL,
		Parameters:      operation.ProvisioningParameters,
	})

	s.provisioningQueue.Add(operation.ID)
	return operation.ID
}

func (s *ProvisioningSuite) CreateUnsuspension(options RuntimeOptions) string {
	provisioningParameters := internal.ProvisioningParameters{
		PlanID: options.ProvidePlanID(),
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    options.ProvideSubAccountID(),
			ServiceManager: &internal.ServiceManagerEntryDTO{
				URL: "sm_url",
				Credentials: internal.ServiceManagerCredentials{
					BasicAuth: internal.ServiceManagerBasicAuth{
						Username: "sm_username",
						Password: "sm_password",
					},
				},
			},
		},
		PlatformRegion: options.ProvidePlatformRegion(),
		Parameters: internal.ProvisioningParametersDTO{
			Region: options.ProvideRegion(),
		},
	}

	operation, err := internal.NewProvisioningOperationWithID(operationID, instanceID, provisioningParameters)
	operation.State = orchestration.Pending
	// in the real processing the URL is set in the handler
	operation.DashboardURL = dashboardURL
	require.NoError(s.t, err)

	err = s.storage.Operations().InsertProvisioningOperation(operation)
	require.NoError(s.t, err)

	instance := &internal.Instance{
		InstanceID:      instanceID,
		GlobalAccountID: globalAccountID,
		SubAccountID:    "dummy-sa",
		ServiceID:       provisioningParameters.ServiceID,
		ServiceName:     broker.KymaServiceName,
		ServicePlanID:   provisioningParameters.PlanID,
		ServicePlanName: broker.AzurePlanName,
		DashboardURL:    dashboardURL,
		Parameters:      operation.ProvisioningParameters,
	}
	err = s.storage.Instances().Insert(*instance)

	suspensionOp := internal.NewSuspensionOperationWithID("susp-id", instance)
	suspensionOp.CreatedAt = time.Now().AddDate(0, 0, -10)
	suspensionOp.State = domain.Succeeded
	s.storage.Operations().InsertDeprovisioningOperation(suspensionOp)

	s.provisioningQueue.Add(operation.ID)
	return operation.ID
}

func (s *ProvisioningSuite) WaitForProvisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetProvisioningOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *ProvisioningSuite) FinishProvisioningOperationByProvisioner(operationID string) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, op.RuntimeID)
}

func (s *ProvisioningSuite) AssertProvisionerStartedProvisioning(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.storage.Operations().GetProvisioningOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var status gqlschema.OperationStatus
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status = s.provisionerClient.FindOperationByRuntimeIDAndType(provisioningOp.RuntimeID, gqlschema.OperationTypeProvision)
		if status.ID != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, gqlschema.OperationStateInProgress, status.State)
}

func (s *ProvisioningSuite) AssertAllStagesFinished(operationID string) {
	operation, _ := s.storage.Operations().GetProvisioningOperationByID(operationID)
	steps := s.provisioningManager.GetAllStages()
	for _, stage := range steps {
		assert.True(s.t, operation.IsStageFinished(stage))
	}
}

func (s *ProvisioningSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *ProvisioningSuite) AssertProvisioningRequest() {
	input := s.fetchProvisionInput()

	labels := *input.RuntimeInput.Labels
	assert.Equal(s.t, instanceID, labels["broker_instance_id"])
	assert.Contains(s.t, labels, "global_subaccount_id")
	assert.NotEmpty(s.t, input.ClusterConfig.GardenerConfig.Name)
}

func (s *ProvisioningSuite) AssertKymaProfile(expectedProfile gqlschema.KymaProfile) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, expectedProfile, *input.KymaConfig.Profile)
}

func (s *ProvisioningSuite) AssertProvider(provider string) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, provider, input.ClusterConfig.GardenerConfig.Provider)
}

func (s *ProvisioningSuite) fetchProvisionInput() gqlschema.ProvisionRuntimeInput {
	input := s.provisionerClient.GetProvisionRuntimeInput(0)
	return input
}

func (s *ProvisioningSuite) AssertMinimalNumberOfNodes(nodes int) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, nodes, input.ClusterConfig.GardenerConfig.AutoScalerMin)
}

func (s *ProvisioningSuite) AssertMaximumNumberOfNodes(nodes int) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, nodes, input.ClusterConfig.GardenerConfig.AutoScalerMax)
}

func (s *ProvisioningSuite) AssertMachineType(machineType string) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, machineType, input.ClusterConfig.GardenerConfig.MachineType)
}

func (s *ProvisioningSuite) AssertOverrides(overrides gqlschema.ConfigEntryInput) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, overrides.Key, input.KymaConfig.Configuration[0].Key)
	assert.Equal(s.t, overrides.Value, input.KymaConfig.Configuration[0].Value)
}

func (s *ProvisioningSuite) AssertZonesCount(zonesCount *int, planID string) {
	input := s.fetchProvisionInput()

	switch planID {
	case broker.AzureHAPlanID:
		if zonesCount != nil {
			// zonesCount was provided in provisioning request
			assert.Equal(s.t, *zonesCount, len(input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones))
			break
		}
		// zonesCount was not provided, should use default value
		assert.Equal(s.t, provider.DefaultAzureHAZonesCount, len(input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AzureConfig.Zones))
	case broker.AWSHAPlanID:
		if zonesCount != nil {
			assert.Equal(s.t, *zonesCount, len(input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones))
			break
		}
		assert.Equal(s.t, provider.DefaultAWSHAZonesCount, len(input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones))
	default:
	}
}

func (s *ProvisioningSuite) AssertSubscription(shared bool, ht hyperscaler.Type) {
	input := s.fetchProvisionInput()
	secretName := input.ClusterConfig.GardenerConfig.TargetSecret
	if shared {
		assert.Equal(s.t, sharedSubscription(ht), secretName)
	} else {
		assert.Equal(s.t, regularSubscription(ht), secretName)
	}
}

func (s *ProvisioningSuite) AssertOIDC(oidcConfig gqlschema.OIDCConfigInput) {
	input := s.fetchProvisionInput()

	assert.Equal(s.t, &oidcConfig, input.ClusterConfig.GardenerConfig.OidcConfig)
}

func (s *ProvisioningSuite) AssertRuntimeAdmins(admins []string) {
	input := s.fetchProvisionInput()
	currentAdmins := input.ClusterConfig.Administrators

	assert.ElementsMatch(s.t, currentAdmins, admins)
}

func regularSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("regular-%s", ht)
}

func sharedSubscription(ht hyperscaler.Type) string {
	return fmt.Sprintf("shared-%s", ht)
}

func (s *ProvisioningSuite) MarkDirectorWithConsoleURL(operationID string) {
	op, err := s.storage.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)
	s.directorClient.SetConsoleURL(op.RuntimeID, op.DashboardURL)
}

func fixConfig() *Config {
	return &Config{
		DbInMemory:                         true,
		DisableProcessOperationsInProgress: false,
		DevelopmentMode:                    true,
		DumpProvisionerRequests:            true,
		OperationTimeout:                   2 * time.Minute,
		Provisioner: input.Config{
			ProvisioningTimeout:   2 * time.Minute,
			DeprovisioningTimeout: 2 * time.Minute,
		},
		Director: director.Config{},
		Database: storage.Config{
			SecretKey: dbSecretKey,
		},
		KymaVersion:        "1.21",
		KymaPreviewVersion: "2.0",

		EnableOnDemandVersion: true,
		Broker: broker.Config{
			EnablePlans: []string{"azure", "trial", "preview"},
		},
		Avs: avs.Config{},
		IAS: ias.Config{
			IdentityProvider: ias.FakeIdentityProviderName,
		},
		AuditLog: auditlog.Config{
			URL:           "https://host1:8080/aaa/v2/",
			User:          "fooUser",
			Password:      "barPass",
			Tenant:        "fooTen",
			EnableSeqHttp: true,
		},
		Monitoring: monitoring.Config{
			Namespace:       "mornitoring",
			ChartUrl:        "notEmptyChart",
			RemoteWriteUrl:  "notEmptyUrl",
			RemoteWritePath: "notEmptyPath",
			Disabled:        false,
		},
		FreemiumProviders:       []string{"aws", "azure"},
		UpdateProcessingEnabled: true,
		Gardener: gardener.Config{
			Project:     "kyma",
			ShootDomain: "sap.com",
		},
	}
}

func fixAccountProvider() *hyperscalerautomock.AccountProvider {
	accountProvider := hyperscalerautomock.AccountProvider{}

	accountProvider.On("GardenerSecretName", mock.Anything, mock.Anything).Return(
		func(ht hyperscaler.Type, tn string) string { return regularSubscription(ht) }, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.Azure).Return(
		func(ht hyperscaler.Type) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("GardenerSharedSecretName", hyperscaler.AWS).Return(
		func(ht hyperscaler.Type) string { return sharedSubscription(ht) }, nil)

	accountProvider.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.Azure, mock.Anything).Return(nil)
	return &accountProvider
}

func fixServiceManagerFactory() provisioning.SMClientFactory {
	smcf := servicemanager.NewFakeServiceManagerClientFactory([]types.ServiceOffering{{
		ID:        "id-001",
		Name:      "xsuaa",
		CatalogID: "off-cat-id-001",
		BrokerID:  brokerID,
	},
		{
			ID:        "connectivity-oferring-id",
			Name:      provisioning.ConnectivityOfferingName,
			CatalogID: "connectivity-service-id",
			BrokerID:  brokerID,
		},
	}, []types.ServicePlan{{
		ID:        "xsuaa-plan-id",
		Name:      "application",
		CatalogID: "xsuaa",
	},
		{
			ID:        "connectivity-plan-id",
			Name:      provisioning.ConnectivityPlanName,
			CatalogID: provisioning.ConnectivityPlanName,
		},
	})
	smcf.SynchronousProvisioning()

	return smcf
}

func createInMemFS() (afero.Fs, error) {

	inMemoryFs := afero.NewMemMapFs()

	fileScript := `
		func myScript() {
		foo: sub_account_id
		bar: tenant_id
		return "fooBar"
	}`

	err := afero.WriteFile(inMemoryFs, "/auditlog-script/script", []byte(fileScript), 0755)
	return inMemoryFs, err
}
