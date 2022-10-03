package provisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestExternalEvalStep_Run(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()
	_, externalEvalCreator, mockOauthServer, mockAvsSvc := setupAvs(t, memoryStorage.Operations())
	defer mockAvsSvc.server.Close()
	defer mockOauthServer.Close()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	operation.Avs.AvsEvaluationInternalId = fixAvsEvaluationInternalId
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)
	step := ExternalEvalStep{
		externalEvalCreator: externalEvalCreator,
	}

	// when
	_, retry, err := step.Run(operation, logrus.New())

	// then
	assert.Zero(t, retry)
	assert.NoError(t, err)
	inDB, _ := memoryStorage.Operations().GetOperationByID(operation.ID)
	assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AVSEvaluationExternalId)
}
