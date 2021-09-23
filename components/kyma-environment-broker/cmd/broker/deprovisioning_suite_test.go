package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/edp"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/event"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ias"
	monitoringmocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/monitoring/mocks"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/deprovisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	subAccountID   = "fakeSubAccountID"
	edpEnvironment = "test"
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

	smcf := servicemanager.NewFakeServiceManagerClientFactory(nil, nil)

	iasFakeClient := ias.NewFakeClient()
	bundleBuilder := ias.NewBundleBuilder(iasFakeClient, cfg.IAS)

	edpClient := fixEDPClient()

	monitoringClient := &monitoringmocks.Client{}
	monitoringClient.On("IsPresent", mock.Anything).Return(true, nil)
	monitoringClient.On("UninstallRelease", mock.Anything).Return(nil, nil)

	accountProvider := fixAccountProvider()

	deprovisionManager := deprovisioning.NewManager(db.Operations(), eventBroker, logs.WithField("deprovisioning", "manager"))

	deprovisioningQueue := NewDeprovisioningProcessingQueue(ctx, workersAmount, deprovisionManager, cfg, db, eventBroker,
		provisionerClient, avsDel, internalEvalAssistant, externalEvalAssistant, smcf,
		bundleBuilder, edpClient, monitoringClient, accountProvider, logs,
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

	require.NoError(s.t, s.storage.Instances().Insert(instance))
	require.NoError(s.t, s.storage.Operations().InsertProvisioningOperation(provisioningOperation))

	state, err := s.provisionerClient.ProvisionRuntime(options.ProvideGlobalAccountID(), options.ProvideSubAccountID(), gqlschema.ProvisionRuntimeInput{})
	require.NoError(s.t, err)

	s.finishProvisioningOperationByProvisioner(gqlschema.OperationTypeProvision, *state.RuntimeID)

	return instance.InstanceID
}

func (s *DeprovisioningSuite) finishProvisioningOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
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

func (s *DeprovisioningSuite) CreateDeprovisioning(instanceId string) string {
	instance, err := s.storage.Instances().GetByID(instanceId)
	require.NoError(s.t, err)

	operation, err := internal.NewDeprovisioningOperationWithID(deprovisioningOpID, instance)
	require.NoError(s.t, err)

	operation.ProvisioningParameters.ErsContext.SubAccountID = subAccountID

	err = s.storage.Operations().InsertDeprovisioningOperation(operation)
	require.NoError(s.t, err)

	s.deprovisioningQueue.Add(operation.ID)

	return operation.ID
}

func (s *DeprovisioningSuite) WaitForDeprovisioningState(operationID string, state domain.LastOperationState) {
	var op *internal.DeprovisioningOperation
	err := wait.PollImmediate(pollingInterval, 2*time.Second, func() (done bool, err error) {
		op, _ = s.storage.Operations().GetDeprovisioningOperationByID(operationID)
		return op.State == state, nil
	})
	assert.NoError(s.t, err, "timeout waiting for the operation expected state %s. The existing operation %+v", state, op)
}

func (s *DeprovisioningSuite) AssertProvisionerStartedDeprovisioning(operationID string) {
	var deprovisioningOp *internal.DeprovisioningOperation
	err := wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		op, err := s.storage.Operations().GetDeprovisioningOperationByID(operationID)
		assert.NoError(s.t, err)
		if op.ProvisionerOperationID != "" {
			deprovisioningOp = op
			return true, nil
		}
		return false, nil
	})
	assert.NoError(s.t, err)

	var status gqlschema.OperationStatus
	err = wait.Poll(pollingInterval, 2*time.Second, func() (bool, error) {
		status = s.provisionerClient.FindOperationByRuntimeIDAndType(deprovisioningOp.RuntimeID, gqlschema.OperationTypeDeprovision)
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

	s.finishOperationByProvisioner(gqlschema.OperationTypeDeprovision, op.RuntimeID)
}

func (s *DeprovisioningSuite) finishOperationByProvisioner(operationType gqlschema.OperationType, runtimeID string) {
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
	}

	for _, key := range metadataTenantKeys {
		client.CreateMetadataTenant(subAccountID, edpEnvironment, edp.MetadataTenantPayload{
			Key:   key,
			Value: "-",
		})
	}

	return client
}
