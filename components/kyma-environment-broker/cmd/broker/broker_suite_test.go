package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"sort"
	"testing"
	"time"

	"code.cloudfoundry.org/lager"
	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerFake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	monitoringmocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring/mocks"
	kebOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	kebRuntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fixedGardenerNamespace = "garden-test"

// BrokerSuiteTest is a helper which allows to write simple tests of any KEB processes (provisioning, deprovisioning, update).
// The starting point of a test could be an HTTP call to Broker API.
type BrokerSuiteTest struct {
	db                storage.BrokerStorage
	provisionerClient *provisioner.FakeClient
	directorClient    *director.FakeClient
	reconcilerClient  *reconciler.FakeClient
	gardenerClient    *gardenerFake.Clientset

	httpServer *httptest.Server
	router     *mux.Router

	t                   *testing.T
	inputBuilderFactory input.CreatorForPlan
}

func (s *BrokerSuiteTest) TearDown() {
	s.httpServer.Close()
}

func NewBrokerSuiteTest(t *testing.T) *BrokerSuiteTest {
	ctx := context.Background()
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	additionalKymaVersions := []string{"1.19", "1.20", "main", "2.0"}
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(defaultKymaVer, additionalKymaVersions)...).Build()
	cfg := fixConfig()

	optionalComponentsDisablers := kebRuntime.ComponentsDisablers{}
	optComponentsSvc := kebRuntime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := kebRuntime.NewDisabledComponentsProvider()

	installerYAML := kebRuntime.ReadYAMLFromFile(t, "kyma-installer-cluster.yaml")
	componentsYAML := kebRuntime.ReadYAMLFromFile(t, "kyma-components.yaml")
	fakeHTTPClient := kebRuntime.NewTestClient(t, installerYAML, componentsYAML, http.StatusOK)

	componentListProvider := kebRuntime.NewComponentsListProvider(
		path.Join("testdata", "managed-runtime-components.yaml"),
		path.Join("testdata", "additional-runtime-components.yaml")).WithHTTPClient(fakeHTTPClient)

	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentListProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
		DefaultTrialProvider:        internal.AWS,
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe", "cf-us10": "us"}, cfg.FreemiumProviders, defaultOIDCValues())

	db := storage.NewMemoryStorage()

	require.NoError(t, err)

	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)

	provisionerClient := provisioner.NewFakeClient()
	eventBroker := event.NewPubSub(logs)

	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, cfg.KymaPreviewVersion, accountVersionMapping)

	directorClient := director.NewFakeClient()
	avsDel, externalEvalCreator, internalEvalUpdater, internalEvalAssistant, externalEvalAssistant := createFakeAvsDelegator(t, db, cfg)

	smcf := fixServiceManagerFactory()
	iasFakeClient := ias.NewFakeClient()
	reconcilerClient := reconciler.NewFakeClient()
	bundleBuilder := ias.NewBundleBuilder(iasFakeClient, cfg.IAS)
	edpClient := edp.NewFakeClient()
	accountProvider := fixAccountProvider()
	inMemoryFs, err := createInMemFS()
	require.NoError(t, err)

	monitoringClient := &monitoringmocks.Client{}
	monitoringClient.On("IsDeployed", mock.Anything).Return(false, nil)
	monitoringClient.On("IsPresent", mock.Anything).Return(false, nil)
	monitoringClient.On("InstallRelease", mock.Anything).Return(nil, nil)
	monitoringClient.On("UninstallRelease", mock.Anything).Return(nil, nil)

	// TODO put Reconciler client in the queue for steps
	provisionManager := provisioning.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(context.Background(), provisionManager, workersAmount, cfg, db, provisionerClient,
		directorClient, inputFactory, avsDel, internalEvalAssistant, externalEvalCreator, internalEvalUpdater, runtimeVerConfigurator,
		runtimeOverrides, smcf, bundleBuilder, edpClient, monitoringClient, accountProvider, inMemoryFs, reconcilerClient, logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	updateManager := update.NewManager(db.Operations(), eventBroker, time.Hour, logs)
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager, 1, db, inputFactory, provisionerClient, eventBroker, logs)
	updateQueue.SpeedUp(10000)
	updateManager.SpeedUp(10000)

	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))
	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, avsDel, internalEvalAssistant, externalEvalAssistant, smcf,
		bundleBuilder, edpClient, monitoringClient, accountProvider, reconcilerClient, logs,
	)

	deprovisioningQueue.SpeedUp(10000)

	gardenerClient := gardenerFake.NewSimpleClientset()
	ts := &BrokerSuiteTest{
		db:                  db,
		provisionerClient:   provisionerClient,
		directorClient:      directorClient,
		reconcilerClient:    reconcilerClient,
		gardenerClient:      gardenerClient,
		router:              mux.NewRouter(),
		t:                   t,
		inputBuilderFactory: inputFactory,
	}

	ts.CreateAPI(inputFactory, cfg, db, provisioningQueue, deprovisioningQueue, updateQueue, logs)

	upgradeEvaluationManager := avs.NewEvaluationManager(avsDel, avs.Config{})
	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient.CoreV1beta1(), fixedGardenerNamespace, runtimeLister, logs)
	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, &upgrade_kyma.TimeSchedule{
		Retry:              10 * time.Millisecond,
		StatusCheck:        100 * time.Millisecond,
		UpgradeKymaTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeVerConfigurator, runtimeResolver, upgradeEvaluationManager,
		cfg, hyperscaler.NewAccountProvider(nil, nil), reconcilerClient, nil, inMemoryFs, monitoringClient, logs, cli)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 10 * time.Millisecond,
		StatusCheck:           100 * time.Millisecond,
		UpgradeClusterTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, upgradeEvaluationManager, logs, cli, *cfg)

	kymaQueue.SpeedUp(1000)
	clusterQueue.SpeedUp(1000)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, clusterQueue, cfg.MaxPaginationPage, logs)
	orchestrationHandler.AttachRoutes(ts.router)
	ts.httpServer = httptest.NewServer(ts.router)
	return ts
}

func defaultOIDCValues() internal.OIDCConfigDTO {
	return internal.OIDCConfigDTO{
		ClientID:       "clinet-id-oidc",
		GroupsClaim:    "gropups",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RSA256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func defaultOIDCConfig() *gqlschema.OIDCConfigInput {
	return &gqlschema.OIDCConfigInput{
		ClientID:       defaultOIDCValues().ClientID,
		GroupsClaim:    defaultOIDCValues().GroupsClaim,
		IssuerURL:      defaultOIDCValues().IssuerURL,
		SigningAlgs:    defaultOIDCValues().SigningAlgs,
		UsernameClaim:  defaultOIDCValues().UsernameClaim,
		UsernamePrefix: defaultOIDCValues().UsernamePrefix,
	}
}

func (s *BrokerSuiteTest) ChangeDefaultTrialProvider(provider internal.CloudProvider) {
	s.inputBuilderFactory.(*input.InputBuilderFactory).SetDefaultTrialProvider(provider)
}

func (s *BrokerSuiteTest) CallAPI(method string, path string, body string) *http.Response {
	cli := s.httpServer.Client()
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", s.httpServer.URL, path), bytes.NewBuffer([]byte(body)))
	req.Header.Set("X-Broker-API-Version", "2.15")
	require.NoError(s.t, err)

	resp, err := cli.Do(req)
	require.NoError(s.t, err)
	return resp
}

func (s *BrokerSuiteTest) CreateAPI(inputFactory broker.PlanValidator, cfg *Config, db storage.BrokerStorage, provisioningQueue *process.Queue, deprovisionQueue *process.Queue, updateQueue *process.Queue, logs logrus.FieldLogger) {
	servicesConfig := map[string]broker.Service{
		broker.KymaServiceName: {
			Description: "",
			Metadata: broker.ServiceMetadata{
				DisplayName: "kyma",
				SupportUrl:  "https://kyma-project.io",
			},
			Plans: map[string]broker.PlanData{
				broker.AzurePlanID: {
					Description: broker.AzurePlanName,
					Metadata:    broker.PlanMetadata{},
				},
				broker.PreviewPlanID: {
					Description: broker.PreviewPlanName,
					Metadata:    broker.PlanMetadata{},
				},
			},
		},
	}
	planDefaults := func(planID string, platformProvider internal.CloudProvider, provider *internal.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	createAPI(s.router, servicesConfig, inputFactory, cfg, db, provisioningQueue, deprovisionQueue, updateQueue, lager.NewLogger("api"), logs, planDefaults)

	s.httpServer = httptest.NewServer(s.router)
}

func createFakeAvsDelegator(t *testing.T, db storage.BrokerStorage, cfg *Config) (*avs.Delegator, *provisioning.ExternalEvalCreator, *provisioning.InternalEvalUpdater, *avs.InternalEvalAssistant, *avs.ExternalEvalAssistant) {
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

	return avsDel, externalEvalCreator, internalEvalUpdater, internalEvalAssistant, externalEvalAssistant
}

func (s *BrokerSuiteTest) CreateProvisionedRuntime(options RuntimeOptions) string {
	randomInstanceId := uuid.New().String()

	instance := fixture.FixInstance(randomInstanceId)
	instance.GlobalAccountID = options.ProvideGlobalAccountID()
	instance.SubAccountID = options.ProvideSubAccountID()
	instance.InstanceDetails.SubAccountID = options.ProvideSubAccountID()
	instance.Parameters.PlatformRegion = options.ProvidePlatformRegion()
	instance.Parameters.Parameters.Region = options.ProvideRegion()
	instance.ProviderRegion = *options.ProvideRegion()

	provisioningOperation := fixture.FixProvisioningOperation(operationID, randomInstanceId)

	require.NoError(s.t, s.db.Instances().Insert(instance))
	require.NoError(s.t, s.db.Operations().InsertProvisioningOperation(provisioningOperation))

	return instance.InstanceID
}

func (s *BrokerSuiteTest) WaitForProvisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, err = s.db.Operations().GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *BrokerSuiteTest) WaitForOperationState(operationID string, state domain.LastOperationState) {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, err = s.db.Operations().GetOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *BrokerSuiteTest) WaitForLastOperation(iid string, state domain.LastOperationState) string {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetLastOperation(iid)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)

	return op.ID
}

func (s *BrokerSuiteTest) FinishProvisioningOperationByProvisioner(operationID string) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, op.RuntimeID)
}

func (s *BrokerSuiteTest) FinishDeprovisioningOperationByProvisioner(operationID string) {
	var op *internal.DeprovisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, err = s.db.Operations().GetDeprovisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	err = s.gardenerClient.CoreV1beta1().Shoots(fixedGardenerNamespace).Delete(context.Background(), op.ShootName, v1.DeleteOptions{})
	require.NoError(s.t, err)

	s.finishOperationByProvisioner(gqlschema.OperationTypeDeprovision, op.RuntimeID)
}

func (s *BrokerSuiteTest) FinishUpdatingOperationByProvisioner(operationID string) {
	var op *internal.UpdatingOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetUpdatingOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgradeShoot, op.RuntimeID)
}

func (s *BrokerSuiteTest) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
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

func (s *BrokerSuiteTest) FinishProvisioningOperationByReconciler(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconciler.State
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(provisioningOp.RuntimeID, provisioningOp.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(provisioningOp.RuntimeID, provisioningOp.ClusterConfigurationVersion, reconciler.ReadyStatus)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertProvisionerStartedProvisioning(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
		if err != nil {
			return false, nil
		}
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

func (s *BrokerSuiteTest) FinishUpgradeKymaOperationByReconciler(operationID string) {
	var upgradeOp *internal.UpgradeKymaOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetUpgradeKymaOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.ClusterConfigurationVersion != 0 {
			upgradeOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconciler.State
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(upgradeOp.InstanceDetails.RuntimeID, upgradeOp.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(upgradeOp.InstanceDetails.RuntimeID, upgradeOp.ClusterConfigurationVersion, reconciler.ReadyStatus)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertReconcilerStartedReconcilingWhenProvisioning(provisioningOpID string) {
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Minute, func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByID(provisioningOpID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconciler.State
	err = wait.Poll(pollingInterval, 20*time.Minute, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(provisioningOp.RuntimeID, 1)
		if state.Cluster != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, reconciler.ReconcilePendingStatus, state.Status)
}

func (s *BrokerSuiteTest) AssertReconcilerStartedReconcilingWhenUpgrading(instanceID string) {
	// wait until UpgradeOperation reaches Apply_Cluster_Configuration step
	var upgradeKymaOp *internal.UpgradeKymaOperation
	err := wait.Poll(pollingInterval, 2*time.Minute, func() (bool, error) {
		op, err := s.db.Operations().GetUpgradeKymaOperationByInstanceID(instanceID)
		if err != nil {
			return false, nil
		}
		if op.InstanceDetails.ClusterConfigurationVersion != 0 {
			upgradeKymaOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconciler.State
	err = wait.Poll(pollingInterval, 20*time.Minute, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(upgradeKymaOp.InstanceDetails.RuntimeID, upgradeKymaOp.InstanceDetails.ClusterConfigurationVersion)
		if state.Cluster != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, reconciler.ReconcilePendingStatus, state.Status)
}

func (s *BrokerSuiteTest) MarkDirectorWithConsoleURL(operationID string) {
	op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)
	s.directorClient.SetConsoleURL(op.RuntimeID, op.DashboardURL)
}

func (s *BrokerSuiteTest) DecodeOperationID(resp *http.Response) string {
	m, err := ioutil.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	var provisioningResp struct {
		Operation string `json:"operation"`
	}
	err = json.Unmarshal(m, &provisioningResp)
	require.NoError(s.t, err)

	return provisioningResp.Operation
}

func (s *BrokerSuiteTest) DecodeOrchestrationID(resp *http.Response) string {
	m, err := ioutil.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	var upgradeResponse orchestration.UpgradeResponse
	err = json.Unmarshal(m, &upgradeResponse)
	require.NoError(s.t, err)

	return upgradeResponse.OrchestrationID
}

func (s *BrokerSuiteTest) DecodeLastUpgradeKymaOperationIDFromOrchestration(resp *http.Response) (string, error) {
	m, err := ioutil.ReadAll(resp.Body)
	s.Log(string(m))
	require.NoError(s.t, err)
	var operationsList orchestration.OperationResponseList
	err = json.Unmarshal(m, &operationsList)
	require.NoError(s.t, err)

	if operationsList.TotalCount == 0 || len(operationsList.Data) == 0 {
		return "", errors.New("no operations found for given orchestration")
	}

	return operationsList.Data[len(operationsList.Data)-1].OperationID, nil
}

func (s *BrokerSuiteTest) AssertShootUpgrade(operationID string, config gqlschema.UpgradeShootInput) {
	// wait until the operation reaches the call to a Provisioner (provisioner operation ID is stored)
	var provisioningOp *internal.UpdatingOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetUpdatingOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var shootUpgrade gqlschema.UpgradeShootInput
	var found bool
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		shootUpgrade, found = s.provisionerClient.LastShootUpgrade(provisioningOp.RuntimeID)
		if found {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	assert.Equal(s.t, config, shootUpgrade)
}

func (s *BrokerSuiteTest) AssertInstanceRuntimeAdmins(instanceId string, expectedAdmins []string) {
	var instance *internal.Instance
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		instance = s.GetInstance(instanceId)
		if instance != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, expectedAdmins, instance.Parameters.Parameters.RuntimeAdministrators)
}

func (s *BrokerSuiteTest) fetchProvisionInput() gqlschema.ProvisionRuntimeInput {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	return input
}

func (s *BrokerSuiteTest) AssertProvider(expectedProvider string) {
	input := s.fetchProvisionInput()
	assert.Equal(s.t, expectedProvider, input.ClusterConfig.GardenerConfig.Provider)
}

func (s *BrokerSuiteTest) AssertProvisionRuntimeInputWithoutKymaConfig() {
	input := s.fetchProvisionInput()
	assert.Nil(s.t, input.KymaConfig)
}

func (s *BrokerSuiteTest) AssertClusterState(operationID string, expectedState reconciler.State) {
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			provisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconciler.State
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetLatestCluster(provisioningOp.RuntimeID)
		if err == nil {
			return true, nil
		}
		return false, err
	})
	assert.NoError(s.t, err)

	assert.Equal(s.t, expectedState, state)
}

func (s *BrokerSuiteTest) AssertClusterConfig(operationID string, expectedClusterConfig *reconciler.Cluster) {
	clusterConfig := s.getClusterConfig(operationID)

	assert.Equal(s.t, *expectedClusterConfig, clusterConfig)
}

func (s *BrokerSuiteTest) AssertClusterKymaConfig(operationID string, expectedKymaConfig reconciler.KymaConfig) {
	clusterConfig := s.getClusterConfig(operationID)

	// values in arrays need to be sorted, because globalOverrides are coming from a map and map's elements' order is not deterministic
	for _, component := range clusterConfig.KymaConfig.Components {
		sort.Slice(component.Configuration, func(i, j int) bool {
			return component.Configuration[i].Key < component.Configuration[j].Key
		})
	}
	for _, component := range expectedKymaConfig.Components {
		sort.Slice(component.Configuration, func(i, j int) bool {
			return component.Configuration[i].Key < component.Configuration[j].Key
		})
	}

	assert.Equal(s.t, expectedKymaConfig, clusterConfig.KymaConfig)
}

func (s *BrokerSuiteTest) AssertClusterConfigWithKubeconfig(id string) {
	clusterConfig := s.getClusterConfig(id)

	assert.NotEmpty(s.t, clusterConfig.Kubeconfig)
}

func (s *BrokerSuiteTest) AssertClusterMetadata(id string, metadata reconciler.Metadata) {
	clusterConfig := s.getClusterConfig(id)

	assert.Equal(s.t, metadata, clusterConfig.Metadata)
}

func (s *BrokerSuiteTest) getClusterConfig(operationID string) reconciler.Cluster {
	provisioningOp, err := s.db.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)

	var clusterConfig *reconciler.Cluster
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		clusterConfig, err = s.reconcilerClient.LastClusterConfig(provisioningOp.RuntimeID)
		if err != nil {
			return false, err
		}
		if clusterConfig.Cluster != "" {
			return true, nil
		}
		return false, nil
	})
	require.NoError(s.t, err)

	return *clusterConfig
}

func (s *BrokerSuiteTest) LastProvisionInput(iid string) gqlschema.ProvisionRuntimeInput {
	// wait until the operation reaches the call to a Provisioner (provisioner operation ID is stored)
	err := wait.Poll(pollingInterval, 4*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetProvisioningOperationByInstanceID(iid)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	return s.provisionerClient.LastProvisioning()
}

func (s *BrokerSuiteTest) Log(msg string) {
	s.t.Log(msg)
}

func (s *BrokerSuiteTest) EnableDumpingProvisionerRequests() {
	s.provisionerClient.EnableRequestDumping()
}

func (s *BrokerSuiteTest) GetInstance(iid string) *internal.Instance {
	inst, err := s.db.Instances().GetByID(iid)
	require.NoError(s.t, err)
	return inst
}

func (s *BrokerSuiteTest) processProvisioningByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)

	s.FinishProvisioningOperationByProvisioner(opID)
	_, err := s.gardenerClient.CoreV1beta1().Shoots(fixedGardenerNamespace).Create(context.Background(), s.fixGardenerShootForOperationID(opID), v1.CreateOptions{})
	require.NoError(s.t, err)

	// simulate the installed fresh Kyma sets the proper label in the Director
	s.MarkDirectorWithConsoleURL(opID)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) fixGardenerShootForOperationID(opID string) *gardenerapi.Shoot {
	op, err := s.db.Operations().GetProvisioningOperationByID(opID)
	require.NoError(s.t, err)

	return &gardenerapi.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      op.ShootName,
			Namespace: fixedGardenerNamespace,
			Labels: map[string]string{
				globalAccountLabel: op.ProvisioningParameters.ErsContext.GlobalAccountID,
				subAccountLabel:    op.ProvisioningParameters.ErsContext.SubAccountID,
			},
			Annotations: map[string]string{
				runtimeIDAnnotation: op.RuntimeID,
			},
		},
		Spec: gardenerapi.ShootSpec{
			Region: "eu",
			Maintenance: &gardenerapi.Maintenance{
				TimeWindow: &gardenerapi.MaintenanceTimeWindow{
					Begin: "030000+0000",
					End:   "040000+0000",
				},
			},
		},
	}
}

func (s *BrokerSuiteTest) processReconcilingByOperationID(opID string) {
	// Provisioner part
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)
	s.FinishProvisioningOperationByProvisioner(opID)

	// Director part
	s.MarkDirectorWithConsoleURL(opID)

	// Reconciler part
	s.AssertReconcilerStartedReconcilingWhenProvisioning(opID)
	s.FinishProvisioningOperationByReconciler(opID)

	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) processProvisioningByInstanceID(iid string) {
	opID := s.WaitForLastOperation(iid, domain.InProgress)

	s.processProvisioningByOperationID(opID)
}

func (s *BrokerSuiteTest) ShootName(id string) string {
	op, err := s.db.Operations().GetProvisioningOperationByID(id)
	require.NoError(s.t, err)
	return op.ShootName
}

func (s *BrokerSuiteTest) AssertAWSRegionAndZone(region string) {
	input := s.provisionerClient.GetLatestProvisionRuntimeInput()
	assert.Equal(s.t, region, input.ClusterConfig.GardenerConfig.Region)
	assert.Contains(s.t, input.ClusterConfig.GardenerConfig.ProviderSpecificConfig.AwsConfig.AwsZones[0].Name, region)
}

// fixExpectedComponentListWithSMProxy provides a fixed components list for Service Management 1.x - when `sm_platform_credentials`
// object is provided: helm-broker, service-catalog, service-catalog-addons and service-manager-proxy components should be installed
func (s *BrokerSuiteTest) fixExpectedComponentListWithSMProxy(opID string) []reconciler.Component {
	return []reconciler.Component{
		{
			URL:       "",
			Component: "ory",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
			},
		},
		{
			URL:       "",
			Component: "monitoring",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
				{
					Key:    "grafana.env.GF_AUTH_GENERIC_OAUTH_CLIENT_ID",
					Value:  "cid",
					Secret: true,
				},
				{
					Key:    "grafana.env.GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET",
					Value:  "csc",
					Secret: true,
				},
			},
		},
		{
			URL:       "",
			Component: "service-catalog",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
			},
		},
		{
			URL:       "",
			Component: "service-catalog-addons",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
			},
		},
		{
			URL:       "",
			Component: "helm-broker",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
			},
		},
		{
			URL:       "",
			Component: "service-manager-proxy",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
				{
					Key:    "config.sm.url",
					Value:  "https://sm.url",
					Secret: false,
				},
				{
					Key:    "sm.user",
					Value:  "smUsername",
					Secret: false,
				},
				{
					Key:    "sm.password",
					Value:  "smPassword",
					Secret: true,
				},
			},
		},
	}
}

// fixExpectedComponentListWithSMOperator provides a fixed components list for Service Management 2.0 - when `sm_operator_credentials`
// object is provided: btp-opeartor component should be installed
func (s *BrokerSuiteTest) fixExpectedComponentListWithSMOperator(opID string) []reconciler.Component {
	return []reconciler.Component{
		{
			URL:       "",
			Component: "ory",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
			},
		},
		{
			URL:       "",
			Component: "monitoring",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
				{
					Key:    "grafana.env.GF_AUTH_GENERIC_OAUTH_CLIENT_ID",
					Value:  "cid",
					Secret: true,
				},
				{
					Key:    "grafana.env.GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET",
					Value:  "csc",
					Secret: true,
				},
			},
		},
		{
			URL:       "",
			Component: "btp-operator",
			Namespace: "kyma-system",
			Configuration: []reconciler.Configuration{
				{
					Key:    "global.domainName",
					Value:  fmt.Sprintf("%s.kyma.sap.com", s.ShootName(opID)),
					Secret: false,
				},
				{
					Key:    "foo",
					Value:  "bar",
					Secret: false,
				},
				{
					Key:    "global.booleanOverride.enabled",
					Value:  false,
					Secret: false,
				},
				{
					Key:    "manager.secret.clientid",
					Value:  "testClientID",
					Secret: true,
				},
				{
					Key:    "manager.secret.clientsecret",
					Value:  "testClientSecret",
					Secret: true,
				},
				{
					Key:    "manager.secret.url",
					Value:  "https://service-manager.kyma.com",
					Secret: false,
				},
				{
					Key:    "manager.secret.tokenurl",
					Value:  "https://test.auth.com",
					Secret: false,
				},
			},
		},
	}
}
