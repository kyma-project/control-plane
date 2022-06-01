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
	"reflect"
	"sort"
	"testing"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"code.cloudfoundry.org/lager"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/notification"
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
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	gardenerClient    dynamic.Interface

	httpServer *httptest.Server
	router     *mux.Router

	t                   *testing.T
	inputBuilderFactory input.CreatorForPlan

	componentProvider componentProviderDecorated
}

type componentProviderDecorated struct {
	componentProvider input.ComponentListProvider
	decorator         map[string]v1alpha1.KymaComponent
}

func (s componentProviderDecorated) AllComponents(kymaVersion internal.RuntimeVersionData) ([]v1alpha1.KymaComponent, error) {
	all, err := s.componentProvider.AllComponents(kymaVersion)
	for i, c := range all {
		if dc, found := s.decorator[c.Name]; found {
			all[i] = dc
		}
	}
	return all, err
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
	decoratedComponentListProvider := componentProviderDecorated{
		componentProvider: componentListProvider,
		decorator:         make(map[string]v1alpha1.KymaComponent),
	}

	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, decoratedComponentListProvider, input.Config{
		MachineImageVersion:         "253",
		KubernetesVersion:           "1.18",
		MachineImage:                "coreos",
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
		DefaultTrialProvider:        internal.AWS,
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe", "cf-us10": "us"}, cfg.FreemiumProviders, defaultOIDCValues())

	db := storage.NewMemoryStorage()

	require.NoError(t, err)

	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)

	gardenerClient := gardener.NewDynamicFakeClient()

	provisionerClient := provisioner.NewFakeClientWithGardener(gardenerClient, "kcp-system")
	eventBroker := event.NewPubSub(logs)

	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, cfg.KymaPreviewVersion, accountVersionMapping, nil)

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

	// TODO put Reconciler client in the queue for steps
	provisionManager := provisioning.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(context.Background(), provisionManager, workersAmount, cfg, db, provisionerClient,
		directorClient, inputFactory, avsDel, internalEvalAssistant, externalEvalCreator, internalEvalUpdater, runtimeVerConfigurator,
		runtimeOverrides, smcf, bundleBuilder, edpClient, accountProvider, inMemoryFs, reconcilerClient, logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	scheme := runtime.NewScheme()
	apiextensionsv1.AddToScheme(scheme)
	fakeK8sSKRClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	updateManager := update.NewManager(db.Operations(), eventBroker, time.Hour, logs)
	rvc := runtimeversion.NewRuntimeVersionConfigurator("", "", nil, db.RuntimeStates())
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager, 1, db, inputFactory, provisionerClient,
		eventBroker, rvc, db.RuntimeStates(), decoratedComponentListProvider, reconcilerClient, *cfg, fakeK8sClientProvider(fakeK8sSKRClient), logs)
	updateQueue.SpeedUp(10000)
	updateManager.SpeedUp(10000)

	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))
	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, avsDel, internalEvalAssistant, externalEvalAssistant, smcf,
		bundleBuilder, edpClient, accountProvider, reconcilerClient, fakeK8sClientProvider(fakeK8sSKRClient), logs,
	)

	deprovisioningQueue.SpeedUp(10000)

	ts := &BrokerSuiteTest{
		db:                  db,
		provisionerClient:   provisionerClient,
		directorClient:      directorClient,
		reconcilerClient:    reconcilerClient,
		gardenerClient:      gardenerClient,
		router:              mux.NewRouter(),
		t:                   t,
		inputBuilderFactory: inputFactory,
		componentProvider:   decoratedComponentListProvider,
	}

	ts.CreateAPI(inputFactory, cfg, db, provisioningQueue, deprovisioningQueue, updateQueue, logs)

	notificationFakeClient := notification.NewFakeClient()
	notificationBundleBuilder := notification.NewBundleBuilder(notificationFakeClient, cfg.Notification)

	upgradeEvaluationManager := avs.NewEvaluationManager(avsDel, avs.Config{})
	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)
	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient, fixedGardenerNamespace, runtimeLister, logs)
	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, &upgrade_kyma.TimeSchedule{
		Retry:              10 * time.Millisecond,
		StatusCheck:        100 * time.Millisecond,
		UpgradeKymaTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeVerConfigurator, runtimeResolver, upgradeEvaluationManager,
		cfg, avs.NewInternalEvalAssistant(cfg.Avs), reconcilerClient, smcf, notificationBundleBuilder, inMemoryFs, logs, cli)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 10 * time.Millisecond,
		StatusCheck:           100 * time.Millisecond,
		UpgradeClusterTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, upgradeEvaluationManager, notificationBundleBuilder, logs, cli, *cfg)

	kymaQueue.SpeedUp(1000)
	clusterQueue.SpeedUp(1000)

	// TODO: in case of cluster upgrade the same Azure Zones must be send to the Provisioner
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, clusterQueue, cfg.MaxPaginationPage, logs)
	orchestrationHandler.AttachRoutes(ts.router)
	ts.httpServer = httptest.NewServer(ts.router)
	return ts
}

func fakeK8sClientProvider(k8sCli client.Client) func(s string) (client.Client, error) {
	return func(s string) (client.Client, error) {
		return k8sCli, nil
	}
}

func defaultOIDCValues() internal.OIDCConfigDTO {
	return internal.OIDCConfigDTO{
		ClientID:       "client-id-oidc",
		GroupsClaim:    "groups",
		IssuerURL:      "https://issuer.url",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}
}

func defaultDNSValues() gardener.DNSProvidersData {
	return gardener.DNSProvidersData{
		Providers: []gardener.DNSProviderData{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_insuite",
				Type:           "route53_type_test",
			},
		},
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
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s != %s. The existing operation %+v", state, op.State, op)
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

func (s *BrokerSuiteTest) FinishProvisioningOperationByProvisioner(operationID string, operationState gqlschema.OperationState) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, operationState, op.RuntimeID)
}

func (s *BrokerSuiteTest) FailProvisioningOperationByProvisioner(operationID string) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(gqlschema.OperationTypeProvision, gqlschema.OperationStateFailed, op.RuntimeID)
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

	err = s.gardenerClient.Resource(gardener.ShootResource).Namespace(fixedGardenerNamespace).Delete(context.Background(), op.ShootName, v1.DeleteOptions{})
	require.NoError(s.t, err)

	s.finishOperationByProvisioner(gqlschema.OperationTypeDeprovision, gqlschema.OperationStateSucceeded, op.RuntimeID)
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
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgradeShoot, gqlschema.OperationStateSucceeded, op.RuntimeID)
}

func (s *BrokerSuiteTest) finishOperationByProvisioner(operationType gqlschema.OperationType, state gqlschema.OperationState, runtimeID string) {
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, state)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *BrokerSuiteTest) MarkClustertConfigurationDeleted(iid string) {
	op, _ := s.db.Operations().GetDeprovisioningOperationByInstanceID(iid)
	s.reconcilerClient.ChangeClusterState(op.RuntimeID, op.ClusterConfigurationVersion, reconcilerApi.StatusDeleted)
}

func (s *BrokerSuiteTest) RemoveFromReconcilerByInstanceID(iid string) {
	op, _ := s.db.Operations().GetDeprovisioningOperationByInstanceID(iid)
	s.reconcilerClient.DeleteCluster(op.RuntimeID)
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

	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(provisioningOp.RuntimeID, provisioningOp.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(provisioningOp.RuntimeID, provisioningOp.ClusterConfigurationVersion, reconcilerApi.StatusReady)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) FinishUpdatingOperationByProvisionerAndReconciler(operationID string) {
	var updatingOp *internal.UpdatingOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetUpdatingOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			updatingOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(updatingOp.RuntimeID, updatingOp.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(updatingOp.RuntimeID, updatingOp.ClusterConfigurationVersion, reconcilerApi.StatusReady)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) FinishUpdatingOperationByReconciler(operationID string) {
	op, err := s.db.Operations().GetUpdatingOperationByID(operationID)
	assert.NoError(s.t, err)
	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(op.RuntimeID, op.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(op.RuntimeID, op.ClusterConfigurationVersion, reconcilerApi.StatusReady)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) FinishUpdatingOperationByReconcilerBoth(operationID string) {
	var updatingOp *internal.UpdatingOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetUpdatingOperationByID(operationID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			updatingOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var state *reconcilerApi.HTTPClusterResponse
	for ccv := updatingOp.ClusterConfigurationVersion; ccv <= updatingOp.ClusterConfigurationVersion+1; ccv++ {
		err = wait.Poll(pollingInterval, 4*time.Second, func() (bool, error) {
			state, err = s.reconcilerClient.GetCluster(updatingOp.RuntimeID, ccv)
			if err != nil {
				return false, err
			}
			if state.Cluster != "" {
				s.reconcilerClient.ChangeClusterState(updatingOp.RuntimeID, ccv, reconcilerApi.StatusReady)
				return true, nil
			}
			return false, nil
		})
		assert.NoError(s.t, err)
	}
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
	err := wait.Poll(pollingInterval, 3*time.Second, func() (bool, error) {
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

	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, 1*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(upgradeOp.InstanceDetails.RuntimeID, upgradeOp.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			s.reconcilerClient.ChangeClusterState(upgradeOp.InstanceDetails.RuntimeID, upgradeOp.ClusterConfigurationVersion, reconcilerApi.StatusReady)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertReconcilerStartedReconcilingWhenProvisioning(provisioningOpID string) {
	var provisioningOp *internal.ProvisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
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

	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(provisioningOp.RuntimeID, 1)
		if state.Cluster != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, reconcilerApi.StatusReconcilePending, state.Status)
}

func (s *BrokerSuiteTest) AssertReconcilerStartedReconcilingWhenUpgrading(instanceID string) {
	// wait until UpgradeOperation reaches Apply_Cluster_Configuration step
	var upgradeKymaOp *internal.Operation
	err := wait.Poll(pollingInterval, time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetLastOperation(instanceID)
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
	assert.NotNil(s.t, upgradeKymaOp)
	var state *reconcilerApi.HTTPClusterResponse
	err = wait.Poll(pollingInterval, time.Second, func() (bool, error) {
		state, err = s.reconcilerClient.GetCluster(upgradeKymaOp.InstanceDetails.RuntimeID, upgradeKymaOp.InstanceDetails.ClusterConfigurationVersion)
		if err != nil {
			return false, err
		}
		if state.Cluster != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, reconcilerApi.StatusReconcilePending, state.Status)
}

func (s *BrokerSuiteTest) DecodeErrorResponse(resp *http.Response) apiresponses.ErrorResponse {
	m, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(s.t, err)

	r := apiresponses.ErrorResponse{}
	err = json.Unmarshal(m, &r)
	require.NoError(s.t, err)

	return r
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
	var provisioningOp *internal.Operation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetOperationByID(operationID)
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

func (s *BrokerSuiteTest) AssertClusterState(operationID string, expectedState reconcilerApi.HTTPClusterResponse) {
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

	var state *reconcilerApi.HTTPClusterResponse
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

func (s *BrokerSuiteTest) AssertClusterConfig(operationID string, expectedClusterConfig *reconcilerApi.Cluster) {
	clusterConfig := s.getClusterConfig(operationID)

	assert.Equal(s.t, *expectedClusterConfig, clusterConfig)
}

func (s *BrokerSuiteTest) AssertClusterKymaConfig(operationID string, expectedKymaConfig reconcilerApi.KymaConfig) {
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

func (s *BrokerSuiteTest) AssertClusterMetadata(id string, metadata reconcilerApi.Metadata) {
	clusterConfig := s.getClusterConfig(id)

	assert.Equal(s.t, metadata, clusterConfig.Metadata)
}

func (s *BrokerSuiteTest) AssertDisabledNetworkFilter(val *bool) {
	var got, exp string
	err := wait.Poll(pollingInterval, 20*time.Second, func() (bool, error) {
		input := s.provisionerClient.GetLatestProvisionRuntimeInput()
		gc := input.ClusterConfig.GardenerConfig
		if reflect.DeepEqual(val, gc.ShootNetworkingFilterDisabled) {
			return true, nil
		}
		got = "<nil>"
		if gc.ShootNetworkingFilterDisabled != nil {
			got = fmt.Sprintf("%v", *gc.ShootNetworkingFilterDisabled)
		}
		exp = "<nil>"
		if val != nil {
			exp = fmt.Sprintf("%v", *val)
		}
		return false, nil
	})
	if err != nil {
		err = fmt.Errorf("ShootNetworkingFilterDisabled expected %v, got %v", exp, got)
	}
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertDisabledNetworkFilterRuntimeState(op string, val *bool) {
	var got, exp string
	err := wait.Poll(pollingInterval, 20*time.Second, func() (bool, error) {
		rs, _ := s.db.RuntimeStates().GetByOperationID(op)
		if reflect.DeepEqual(val, rs.ClusterConfig.ShootNetworkingFilterDisabled) {
			return true, nil
		}
		got = "<nil>"
		if rs.ClusterConfig.ShootNetworkingFilterDisabled != nil {
			got = fmt.Sprintf("%v", *rs.ClusterConfig.ShootNetworkingFilterDisabled)
		}
		exp = "<nil>"
		if val != nil {
			exp = fmt.Sprintf("%v", *val)
		}
		return false, fmt.Errorf("ShootNetworkingFilterDisabled expected %v, got %v", exp, got)
	})
	if err != nil {
		err = fmt.Errorf("ShootNetworkingFilterDisabled expected %v, got %v", exp, got)
	}
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) getClusterConfig(operationID string) reconcilerApi.Cluster {
	provisioningOp, err := s.db.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)

	var clusterConfig *reconcilerApi.Cluster
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		clusterConfig, err = s.reconcilerClient.LastClusterConfig(provisioningOp.RuntimeID)
		if err != nil {
			return false, err
		}
		if clusterConfig.RuntimeID != "" {
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

	s.FinishProvisioningOperationByProvisioner(opID, gqlschema.OperationStateSucceeded)
	_, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(fixedGardenerNamespace).Create(context.Background(), s.fixGardenerShootForOperationID(opID), v1.CreateOptions{})
	require.NoError(s.t, err)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) failProvisioningByOperationID(opID string) {
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)

	s.FinishProvisioningOperationByProvisioner(opID, gqlschema.OperationStateFailed)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Failed)
}

func (s *BrokerSuiteTest) fixGardenerShootForOperationID(opID string) *unstructured.Unstructured {
	op, err := s.db.Operations().GetProvisioningOperationByID(opID)
	require.NoError(s.t, err)

	un := unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      op.ShootName,
				"namespace": fixedGardenerNamespace,
				"labels": map[string]interface{}{
					globalAccountLabel: op.ProvisioningParameters.ErsContext.GlobalAccountID,
					subAccountLabel:    op.ProvisioningParameters.ErsContext.SubAccountID,
				},
				"annotations": map[string]interface{}{
					runtimeIDAnnotation: op.RuntimeID,
				},
			},
			"spec": map[string]interface{}{
				"region": "eu",
				"maintenance": map[string]interface{}{
					"timeWindow": map[string]interface{}{
						"begin": "030000+0000",
						"end":   "040000+0000",
					},
				},
			},
		},
	}
	un.SetGroupVersionKind(shootGVK)
	return &un
}

func (s *BrokerSuiteTest) processReconcilingByOperationID(opID string) {
	// Provisioner part
	s.WaitForProvisioningState(opID, domain.InProgress)
	s.AssertProvisionerStartedProvisioning(opID)
	s.FinishProvisioningOperationByProvisioner(opID, gqlschema.OperationStateSucceeded)
	_, err := s.gardenerClient.Resource(gardener.ShootResource).Namespace(fixedGardenerNamespace).Create(context.Background(), s.fixGardenerShootForOperationID(opID), v1.CreateOptions{})
	require.NoError(s.t, err)

	// Reconciler part
	s.AssertReconcilerStartedReconcilingWhenProvisioning(opID)
	s.FinishProvisioningOperationByReconciler(opID)

	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) processProvisioningByInstanceID(iid string) {
	opID := s.WaitForLastOperation(iid, domain.InProgress)

	s.processProvisioningByOperationID(opID)
}

func (s *BrokerSuiteTest) processReconciliationByInstanceID(iid string) {
	opID := s.WaitForLastOperation(iid, domain.InProgress)

	s.processReconcilingByOperationID(opID)
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
func (s *BrokerSuiteTest) fixExpectedComponentListWithSMProxy(opID string) []reconcilerApi.Component {
	return []reconcilerApi.Component{
		{
			URL:       "",
			Component: "ory",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
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
			Configuration: []reconcilerApi.Configuration{
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
			Component: "service-catalog",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
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
			Configuration: []reconcilerApi.Configuration{
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
			Configuration: []reconcilerApi.Configuration{
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
			URL:       "https://sm-proxy",
			Component: "service-manager-proxy",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
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
func (s *BrokerSuiteTest) fixExpectedComponentListWithSMOperator(opID, smClusterID string) []reconcilerApi.Component {
	return []reconcilerApi.Component{
		{
			URL:       "",
			Component: "ory",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
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
			Configuration: []reconcilerApi.Configuration{
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
			URL:       "https://btp-operator",
			Component: "btp-operator",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
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
					Key:    "manager.secret.sm_url",
					Value:  "https://service-manager.kyma.com",
					Secret: false,
				},
				{
					Key:    "manager.secret.tokenurl",
					Value:  "https://test.auth.com",
					Secret: false,
				},
				{
					Key:    "cluster.id",
					Value:  smClusterID,
					Secret: false,
				},
			},
		},
	}
}

func mockBTPOperatorClusterID() {
	mock := func(string) (string, error) {
		return "cluster_id", nil
	}
	update.ConfigMapGetter = mock
	upgrade_kyma.ConfigMapGetter = mock
}
