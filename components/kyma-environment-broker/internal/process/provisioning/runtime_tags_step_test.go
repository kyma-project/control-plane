package provisioning

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	provisionerAutomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeTagsStep_Run(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()
	internalEvalUpdater, _, mockOauthServer, mockAvsSvc := setupAvs(t, memoryStorage.Operations())
	defer mockAvsSvc.server.Close()
	defer mockOauthServer.Close()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	operation.Avs.AvsEvaluationInternalId = fixAvsEvaluationInternalId
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	provisionerClient := setupProvisionerClient(operation.RuntimeID)

	step := RuntimeTagsStep{
		internalEvalUpdater: internalEvalUpdater,
		provisionerClient:   provisionerClient,
	}

	// when
	_, retry, err := step.Run(operation, logrus.New())

	// then
	assert.Zero(t, retry)
	assert.NoError(t, err)
	inDB, _ := memoryStorage.Operations().GetOperationByID(operation.ID)
	assert.Equal(t, 4, len(mockAvsSvc.evals[inDB.Avs.AvsEvaluationInternalId].Tags))
}

func setupAvs(t *testing.T, operations storage.Operations) (*InternalEvalUpdater, *ExternalEvalCreator, *httptest.Server, *mockAvsService) {
	mockOauthServer := newMockAvsOauthServer()
	mockAvsSvc := newMockAvsService(t, false)
	mockAvsSvc.startServer()
	mockAvsSvc.evals[fixAvsEvaluationInternalId] = fixAvsEvaluation()
	avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
	avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	require.NoError(t, err)
	avsDel := avs.NewDelegator(avsClient, avsConfig, operations)
	internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
	internalEvalUpdater := NewInternalEvalUpdater(avsDel, internalEvalAssistant, avsConfig)
	externalEvalAssistant := avs.NewExternalEvalAssistant(avsConfig)
	externalEvalCreator := NewExternalEvalCreator(avsDel, false, externalEvalAssistant)

	return internalEvalUpdater, externalEvalCreator, mockOauthServer, mockAvsSvc
}

func setupProvisionerClient(runtimeID string) provisioner.Client {
	provisionerClient := &provisionerAutomock.Client{}
	provisionerClient.On("RuntimeStatus", statusGlobalAccountID, runtimeID).Return(gqlschema.RuntimeStatus{
		LastOperationStatus:     nil,
		RuntimeConnectionStatus: nil,
		RuntimeConfiguration: &gqlschema.RuntimeConfig{ClusterConfig: &gqlschema.GardenerConfig{
			Name:   ptr.String("test-gardener-name"),
			Region: ptr.String("test-gardener-region"),
			Seed:   ptr.String("test-gardener-seed"),
		}},
	}, nil)
	return provisionerClient
}
