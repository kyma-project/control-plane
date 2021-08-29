package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.cloudfoundry.org/lager"
	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerFake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	kebOrchestration "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration"
	orchestrate "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/handlers"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_cluster"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/upgrade_kyma"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/update"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	kebRuntime "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeversion"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// BrokerSuiteTest is a helper which allows to write simple tests of any KEB processes (provisioning, deprovisioning, update).
// The starting point of a test could be an HTTP call to Broker API.
type BrokerSuiteTest struct {
	db                storage.BrokerStorage
	provisionerClient *provisioner.FakeClient
	directorClient    *director.FakeClient
	gardenerClient    *gardenerFake.Clientset

	httpServer *httptest.Server
	router     *mux.Router

	t *testing.T
}

func (s *BrokerSuiteTest) TearDown() {
	s.httpServer.Close()
}

func NewBrokerSuiteTest(t *testing.T) *BrokerSuiteTest {
	ctx := context.Background()
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	additionalKymaVersions := []string{"1.19", "1.20", "main"}
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(defaultKymaVer, additionalKymaVersions)...).Build()
	cfg := fixConfig()

	optionalComponentsDisablers := kebRuntime.ComponentsDisablers{}
	optComponentsSvc := kebRuntime.NewOptionalComponentsService(optionalComponentsDisablers)

	disabledComponentsProvider := kebRuntime.NewDisabledComponentsProvider()

	componentListProvider := &automock.ComponentListProvider{}
	componentListProvider.On("AllComponents", mock.Anything).Return([]v1alpha1.KymaComponent{}, nil)

	inputFactory, err := input.NewInputBuilderFactory(optComponentsSvc, disabledComponentsProvider, componentListProvider, input.Config{
		MachineImageVersion:         "coreos",
		KubernetesVersion:           "1.18",
		MachineImage:                "253",
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders, defaultOIDCValues())

	db := storage.NewMemoryStorage()

	require.NoError(t, err)

	logs := logrus.New()
	logs.SetLevel(logrus.DebugLevel)

	provisionerClient := provisioner.NewFakeClient()
	eventBroker := event.NewPubSub(logs)

	runtimeOverrides := runtimeoverrides.NewRuntimeOverrides(ctx, cli)
	accountVersionMapping := runtimeversion.NewAccountVersionMapping(ctx, cli, cfg.VersionConfig.Namespace, cfg.VersionConfig.Name, logs)
	runtimeVerConfigurator := runtimeversion.NewRuntimeVersionConfigurator(cfg.KymaVersion, accountVersionMapping)

	directorClient := director.NewFakeClient()
	avsDel, externalEvalCreator, internalEvalUpdater, internalEvalAssistant, externalEvalAssistant := createFakeAvsDelegator(t, db, cfg)

	smcf := fixServiceManagerFactory()
	iasFakeClient := ias.NewFakeClient()
	bundleBuilder := ias.NewBundleBuilder(iasFakeClient, cfg.IAS)
	edpClient := edp.NewFakeClient()
	accountProvider := fixAccountProvider()
	inMemoryFs, err := createInMemFS()
	require.NoError(t, err)

	provisionManager := provisioning.NewStagedManager(db.Operations(), eventBroker, cfg.OperationTimeout, logs.WithField("provisioning", "manager"))
	provisioningQueue := NewProvisioningProcessingQueue(context.Background(), provisionManager, workersAmount, cfg, db, provisionerClient,
		directorClient, inputFactory, avsDel, internalEvalAssistant, externalEvalCreator, internalEvalUpdater, runtimeVerConfigurator,
		runtimeOverrides, smcf, bundleBuilder, edpClient, accountProvider, inMemoryFs, logs)

	provisioningQueue.SpeedUp(10000)
	provisionManager.SpeedUp(10000)

	updateManager := update.NewManager(db.Operations(), eventBroker, time.Hour, logs)
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager, 1, db, inputFactory, provisionerClient, eventBroker, logs)
	updateQueue.SpeedUp(10000)
	updateManager.SpeedUp(10000)

	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))
	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, avsDel, internalEvalAssistant, externalEvalAssistant, smcf,
		bundleBuilder, edpClient, accountProvider, logs,
	)

	deprovisioningQueue.SpeedUp(10000)

	// create 'kyma/upgrade' endpoint
	upgradeEvaluationManager := avs.NewEvaluationManager(avsDel, avs.Config{})
	gardenerClient := gardenerFake.NewSimpleClientset()
	runtimeLister := kebOrchestration.NewRuntimeLister(db.Instances(), db.Operations(), kebRuntime.NewConverter(defaultRegion), logs)

	runtimeResolver := orchestration.NewGardenerRuntimeResolver(gardenerClient.CoreV1beta1(), "gardener-e2e", runtimeLister, logs)

	kymaQueue := NewKymaOrchestrationProcessingQueue(ctx, db, runtimeOverrides, provisionerClient, eventBroker, inputFactory, &upgrade_kyma.TimeSchedule{
		Retry:              10 * time.Millisecond,
		StatusCheck:        100 * time.Millisecond,
		UpgradeKymaTimeout: 20 * time.Second,
	}, 250*time.Millisecond, runtimeVerConfigurator, runtimeResolver, upgradeEvaluationManager, cfg, smcf, inMemoryFs, logs, cli)

	clusterQueue := NewClusterOrchestrationProcessingQueue(ctx, db, provisionerClient, eventBroker, inputFactory, &upgrade_cluster.TimeSchedule{
		Retry:                 10 * time.Millisecond,
		StatusCheck:           100 * time.Millisecond,
		UpgradeClusterTimeout: 4 * time.Second,
	}, 250*time.Millisecond, runtimeResolver, upgradeEvaluationManager, logs, cli, *cfg)

	kymaQueue.SpeedUp(1000)
	clusterQueue.SpeedUp(1000)
	orchestrationHandler := orchestrate.NewOrchestrationHandler(db, kymaQueue, clusterQueue, cfg.MaxPaginationPage, logs)

	ts := &BrokerSuiteTest{
		db:                db,
		provisionerClient: provisionerClient,
		directorClient:    directorClient,
		gardenerClient:    gardenerClient,
		t:                 t,
		router:            mux.NewRouter(),
	}

	// attach upgrade routes
	orchestrationHandler.AttachRoutes(ts.router)

	// attach OSB API routes
	ts.CreateAPI(inputFactory, cfg, db, provisioningQueue, deprovisioningQueue, updateQueue, logs)
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
				"4deee563-e5ec-4731-b9b1-53b42d855f0c": {
					Description: "azure",
					Metadata:    broker.PlanMetadata{},
				},
			},
		},
	}
	createAPI(s.router, servicesConfig, inputFactory, cfg, db, provisioningQueue, deprovisionQueue, updateQueue, lager.NewLogger("api"), logs)
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

func (s *BrokerSuiteTest) WaitForUpgradeKymaState(instanceID string, state domain.LastOperationState) {
	var op *internal.UpgradeKymaOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, err = s.db.Operations().GetUpgradeKymaOperationByInstanceID(instanceID)
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

func (s *BrokerSuiteTest) WaitForOrchestrationState(orchestrationID string, state string) {
	var orchestration *internal.Orchestration
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		orchestration, _ = s.db.Orchestrations().GetByID(orchestrationID)
		return orchestration.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the orchestration expected state %s. The existing orchestration %+v", state, orchestration)
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

func (s *BrokerSuiteTest) FinishUpgradeKymaOperationByProvisioner(instanceID string) {
	var op *internal.UpgradeKymaOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetUpgradeKymaOperationByInstanceID(instanceID)
		if op.InstanceDetails.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)
	s.finishOperationByProvisioner(gqlschema.OperationTypeUpgrade, op.InstanceDetails.RuntimeID)
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

func (s *BrokerSuiteTest) AssertProvisionerStartedUpgradeKyma(instanceID string) {
	// wait until UpgradeKymaOperation reaches Upgrade_Kyma step
	var upgradeKymaOp *internal.UpgradeKymaOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.db.Operations().GetUpgradeKymaOperationByInstanceID(instanceID)
		if err != nil {
			return false, nil
		}
		if op.ProvisionerOperationID != "" {
			upgradeKymaOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var status gqlschema.OperationStatus
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status = s.provisionerClient.FindOperationByRuntimeIDAndType(upgradeKymaOp.Operation.InstanceDetails.RuntimeID, gqlschema.OperationTypeUpgrade)
		if status.ID != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, gqlschema.OperationStateInProgress, status.State)
}

func (s *BrokerSuiteTest) MarkDirectorWithConsoleURL(operationID string) {
	op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)
	s.directorClient.SetConsoleURL(op.RuntimeID, op.DashboardURL)
}

func (s *BrokerSuiteTest) DecodeOperationID(resp *http.Response) string {
	m, err := ioutil.ReadAll(resp.Body)
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
	var upgradeKymaResp struct {
		OrchestrationID string `json:"orchestrationID"`
	}
	strBody := string(m)
	s.Log(strBody)
	err = json.Unmarshal(m, &upgradeKymaResp)
	require.NoError(s.t, err)
	return upgradeKymaResp.OrchestrationID
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
	// simulate the installed fresh Kyma sets the proper label in the Director
	s.MarkDirectorWithConsoleURL(opID)

	s.CreateShootInGardener(opID)

	// provisioner finishes the operation
	s.WaitForOperationState(opID, domain.Succeeded)
}

func (s *BrokerSuiteTest) processUpgradeKymaByInstanceID(instanceID string) {
	s.WaitForUpgradeKymaState(instanceID, domain.InProgress)
	s.AssertProvisionerStartedUpgradeKyma(instanceID)

	s.FinishUpgradeKymaOperationByProvisioner(instanceID)
}

func (s *BrokerSuiteTest) processProvisioningByInstanceID(iid string) {
	opID := s.WaitForLastOperation(iid, domain.InProgress)

	s.processProvisioningByOperationID(opID)
}

func (s *BrokerSuiteTest) CreateShootInGardener(opID string) {
	provOp, _ := s.db.Provisioning().GetProvisioningOperationByID(opID)
	runtimeID := provOp.RuntimeID

	shoot := &gardenerapi.Shoot{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("shoot%s", runtimeID),
			Namespace: "gardener-e2e",
			Labels: map[string]string{
				globalAccountLabel: globalAccountID,
				subAccountLabel:    subAccountID,
			},
			Annotations: map[string]string{
				runtimeIDAnnotation: runtimeID,
			},
		},
		Spec: gardenerapi.ShootSpec{
			Region: "cf-eu10",
			Maintenance: &gardenerapi.Maintenance{
				TimeWindow: &gardenerapi.MaintenanceTimeWindow{
					Begin: "030000+0000",
					End:   "040000+0000",
				},
			},
		},
	}

	_, err := s.gardenerClient.CoreV1beta1().Shoots("gardener-e2e").Create(context.Background(), shoot, metaV1.CreateOptions{})
	require.NoError(s.t, err)
}

func (s *BrokerSuiteTest) AssertUpgradeKyma(operationID, runtimeID string) {
	provisioningState, _ := s.db.RuntimeStates().GetByOperationID(operationID)
	s.Log(provisioningState.RuntimeID)
	allStates, _ := s.db.RuntimeStates().ListByRuntimeID(runtimeID)
	assert.Equal(s.t, allStates[0].KymaConfig, allStates[1].KymaConfig)
	s.Log(string(rune(len(allStates))))
}
