package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
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
	"io/ioutil"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

type UpdateSuite struct {
	HttpSuite
	db                storage.BrokerStorage
	provisionerClient *provisioner.FakeClient
	directorClient    *director.FakeClient
}

func (s *UpdateSuite) TearDown() {
	s.httpServer.Close()
}

func NewUpdateSuite(t *testing.T) *UpdateSuite {
	ctx := context.Background()
	sch := runtime.NewScheme()
	require.NoError(t, coreV1.AddToScheme(sch))
	additionalKymaVersions := []string{"1.19", "1.20", "main"}
	cli := fake.NewFakeClientWithScheme(sch, fixK8sResources(defaultKymaVer, additionalKymaVersions)...)
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
		Timeout:                     time.Minute,
		URL:                         "http://localhost",
		DefaultGardenerShootPurpose: "testing",
	}, defaultKymaVer, map[string]string{"cf-eu10": "europe"}, cfg.FreemiumProviders)

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
	avsDel, externalEvalCreator, internalEvalUpdater, internalEvalAssistant := createFakeAvsDelegator(t, db, cfg)

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
	updateQueue := NewUpdateProcessingQueue(context.Background(), updateManager,1, db, inputFactory, provisionerClient, eventBroker, logs)
	updateQueue.SpeedUp(10000)
	updateManager.SpeedUp(10000)



	httpSuite := NewHttpSuite(t)
	httpSuite.CreateAPI(inputFactory, cfg, db, provisioningQueue, nil, updateQueue, logs)

	return &UpdateSuite{
		HttpSuite:         httpSuite,
		db:                db,
		provisionerClient: provisionerClient,
		directorClient:    directorClient,
	}
}

func createFakeAvsDelegator(t *testing.T, db storage.BrokerStorage, cfg *Config) (*avs.Delegator, *provisioning.ExternalEvalCreator, *provisioning.InternalEvalUpdater, *avs.InternalEvalAssistant) {
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

	return avsDel, externalEvalCreator, internalEvalUpdater, internalEvalAssistant
}

func (s *UpdateSuite) CreateProvisionedRuntime(options RuntimeOptions) string {
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

	//state, err := s.provisionerClient.ProvisionRuntime(options.ProvideGlobalAccountID(), options.ProvideSubAccountID(), gqlschema.ProvisionRuntimeInput{})
	//require.NoError(s.t, err)
	//
	//s.finishProvisioningOperationByProvisioner(gqlschema.OperationTypeProvision, *state.RuntimeID)

	return instance.InstanceID
}

func (s *UpdateSuite) WaitForProvisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.ProvisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetProvisioningOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *UpdateSuite) WaitForOperationState(operationID string, state domain.LastOperationState) {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 20*time.Second, func() (done bool, err error) {
		op, _ = s.db.Operations().GetOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *UpdateSuite) FinishProvisioningOperationByProvisioner(operationID string) {
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

func (s *UpdateSuite) FinishUpdatingOperationByProvisioner(operationID string) {
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

func (s *UpdateSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
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

func (s *UpdateSuite) AssertProvisionerStartedProvisioning(operationID string) {
	// wait until ProvisioningOperation reaches CreateRuntime step
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

func (s *UpdateSuite) MarkDirectorWithConsoleURL(operationID string) {
	op, err := s.db.Operations().GetProvisioningOperationByID(operationID)
	assert.NoError(s.t, err)
	s.directorClient.SetConsoleURL(op.RuntimeID, op.DashboardURL)
}

func (s *UpdateSuite) DecodeOperationID(resp *http.Response) string {
	m, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(m))
	require.NoError(s.t, err)
	var provisioningResp struct {
		Operation string `json:"operation"`
	}
	json.Unmarshal(m, &provisioningResp)
	return provisioningResp.Operation
}

func (s *UpdateSuite) AssertShootUpgrade(operationID string, config gqlschema.UpgradeShootInput) {
	// wait until ProvisioningOperation reaches CreateRuntime step
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
