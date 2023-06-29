package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	kebConfig "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/reconciler"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	subAccountID    = "fake-subaccount-id"
	badSubAccountID = "bad-fake-subaccount-id"
	edpEnvironment  = "test"
)

type DeprovisioningSuite struct {
	provisionerClient   *provisioner.FakeClient
	deprovisioningQueue *process.Queue
	storage             storage.BrokerStorage

	t *testing.T
}

func NewDeprovisioningSuite(t *testing.T) *DeprovisioningSuite {
	ctx, _ := context.WithTimeout(context.Background(), 20*time.Minute)

	logs := logrus.New()
	logs.Formatter.(*logrus.TextFormatter).TimestampFormat = "15:04:05.000"

	cfg := fixConfig()
	cfg.EDP.Environment = edpEnvironment

	db := storage.NewMemoryStorage()
	eventBroker := event.NewPubSub(logs)
	provisionerClient := provisioner.NewFakeClient()

	server := avs.NewMockAvsServer(t)
	mockServer := avs.FixMockAvsServer(server)
	avsConfig := avs.Config{
		OauthTokenEndpoint: fmt.Sprintf("%s/oauth/token", mockServer.URL),
		ApiEndpoint:        fmt.Sprintf("%s/api/v2/evaluationmetadata", mockServer.URL),
	}
	client, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	assert.NoError(t, err)
	_, err = client.CreateEvaluation(&avs.BasicEvaluationCreateRequest{
		Name: "fake-evaluation",
	})
	assert.NoError(t, err)
	avsDel := avs.NewDelegator(client, avsConfig, db.Operations())
	externalEvalAssistant := avs.NewExternalEvalAssistant(cfg.Avs)
	internalEvalAssistant := avs.NewInternalEvalAssistant(cfg.Avs)

	iasFakeClient := ias.NewFakeClient()
	bundleBuilder := ias.NewBundleBuilder(iasFakeClient, cfg.IAS)

	edpClient := fixEDPClient()
	reconcilerClient := reconciler.NewFakeClient()

	accountProvider := fixAccountProvider()

	deprovisionManager := process.NewStagedManager(db.Operations(), eventBroker, time.Minute, logs.WithField("deprovisioning", "manager"))
	deprovisionManager.SpeedUp(1000)
	scheme := runtime.NewScheme()
	apiextensionsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	fakeK8sSKRClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	sch := internal.NewSchemeForTests()
	cli := fake.NewClientBuilder().WithScheme(sch).WithRuntimeObjects(fixK8sResources(defaultKymaVer, []string{})...).Build()

	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(ctx, cli, logrus.New(), defaultKymaVer),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())

	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, avsDel, internalEvalAssistant, externalEvalAssistant,
		bundleBuilder, edpClient, accountProvider, reconcilerClient, fakeK8sClientProvider(fakeK8sSKRClient), fakeK8sSKRClient, configProvider, logs,
	)

	deprovisioningQueue.SpeedUp(10000)

	return &DeprovisioningSuite{
		provisionerClient:   provisionerClient,
		deprovisioningQueue: deprovisioningQueue,
		storage:             db,

		t: t,
	}
}

func (s *DeprovisioningSuite) CreateProvisionedRuntime(options RuntimeOptions) string {
	randomInstanceId := uuid.New().String()

	instance := fixture.FixInstance(randomInstanceId)
	instance.GlobalAccountID = options.ProvideGlobalAccountID()
	instance.SubAccountID = options.ProvideSubAccountID()
	instance.InstanceDetails.SubAccountID = options.ProvideSubAccountID()
	instance.Parameters.PlatformRegion = options.ProvidePlatformRegion()
	instance.Parameters.Parameters.Region = options.ProvideRegion()
	instance.ProviderRegion = *options.ProvideRegion()

	provisioningOperation := fixture.FixProvisioningOperation(operationID, randomInstanceId)
	provisioningOperation.SubAccountID = options.ProvideSubAccountID()

	require.NoError(s.t, s.storage.Instances().Insert(instance))
	require.NoError(s.t, s.storage.Operations().InsertOperation(provisioningOperation))

	state, err := s.provisionerClient.ProvisionRuntime(options.ProvideGlobalAccountID(), options.ProvideSubAccountID(), gqlschema.ProvisionRuntimeInput{})
	require.NoError(s.t, err)

	s.finishProvisioningOperationByProvisioner(gqlschema.OperationTypeProvision, *state.RuntimeID)

	return instance.InstanceID
}

func (s *DeprovisioningSuite) finishProvisioningOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindOperationByRuntimeIDAndType(runtimeID, operationType)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, gqlschema.OperationStateSucceeded)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *DeprovisioningSuite) CreateDeprovisioning(operationID, instanceId string) string {
	instance, err := s.storage.Instances().GetByID(instanceId)
	require.NoError(s.t, err)

	operation, err := internal.NewDeprovisioningOperationWithID(operationID, instance)
	require.NoError(s.t, err)

	operation.ProvisioningParameters.ErsContext.SubAccountID = subAccountID

	err = s.storage.Operations().InsertDeprovisioningOperation(operation)
	require.NoError(s.t, err)

	s.deprovisioningQueue.Add(operation.ID)

	return operation.ID
}

func (s *DeprovisioningSuite) WaitForDeprovisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.Operation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. %v The existing operation %+v", state, op.State, op)
}

func (s *DeprovisioningSuite) AssertProvisionerStartedDeprovisioning(operationID string) {
	var provisionerOperationID string
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.storage.Operations().GetOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			provisionerOperationID = op.ProvisionerOperationID
			return true, nil
		}
		return false, nil
	})
	require.NoError(s.t, err)

	var status gqlschema.OperationStatus
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status = s.provisionerClient.FindOperationByProvisionerOperationID(provisionerOperationID)
		if status.ID != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)
	assert.Equal(s.t, gqlschema.OperationStateInProgress, status.State)
}

func (s *DeprovisioningSuite) FinishDeprovisioningOperationByProvisioner(operationID string) {
	var op *internal.DeprovisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetDeprovisioningOperationByID(operationID)
		if op.RuntimeID != "" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation with runtimeID. The existing operation %+v", op)

	s.finishOperationByProvisioner(op.ProvisionerOperationID)
}

func (s *DeprovisioningSuite) finishOperationByProvisioner(provisionerOperationID string) {
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status := s.provisionerClient.FindOperationByProvisionerOperationID(provisionerOperationID)
		if status.ID != nil {
			s.provisionerClient.FinishProvisionerOperation(*status.ID, gqlschema.OperationStateSucceeded)
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err, "timeout waiting for provisioner operation to exist")
}

func (s *DeprovisioningSuite) updateSubAccountIDForDeprovisioningOperation(options RuntimeOptions, instanceId string) {
	op, err := s.storage.Operations().GetDeprovisioningOperationByInstanceID(instanceId)
	assert.NoError(s.t, err, "failed to GetDeprovisioningOperationByInstanceID: %v", instanceId)
	op.SubAccountID = options.ProvideSubAccountID()
	_, err = s.storage.Operations().UpdateDeprovisioningOperation(*op)
	assert.NoError(s.t, err, "failed to UpdateDeprovisioningOperation: %v", op)
}

func (s *DeprovisioningSuite) AssertInstanceRemoved(instanceId string) {
	instance, err := s.storage.Instances().GetByID(instanceId)
	assert.Error(s.t, err)
	if dberr.IsNotFound(err) {
		assert.Nil(s.t, instance)
	} else {
		assert.Fail(s.t, "failed to get instance", err)
	}
}

func (s *DeprovisioningSuite) AssertInstanceNotRemoved(instanceId string) {
	instance, err := s.storage.Instances().GetByID(instanceId)
	assert.NoError(s.t, err)
	assert.NotNil(s.t, instance)
}

func fixEDPClient() *edp.FakeClient {
	client := edp.NewFakeClient()
	client.CreateDataTenant(edp.DataTenantPayload{
		Name:        subAccountID,
		Environment: edpEnvironment,
		Secret:      base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", subAccountID, edpEnvironment))),
	})

	metadataTenantKeys := []string{
		edp.MaasConsumerEnvironmentKey,
		edp.MaasConsumerRegionKey,
		edp.MaasConsumerSubAccountKey,
		edp.MaasConsumerServicePlan,
	}

	for _, key := range metadataTenantKeys {
		client.CreateMetadataTenant(subAccountID, edpEnvironment, edp.MetadataTenantPayload{
			Key:   key,
			Value: "-",
		})
	}

	return client
}
