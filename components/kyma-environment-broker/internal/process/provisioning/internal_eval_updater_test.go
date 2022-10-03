package provisioning

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	FixAvsEvaluationInternalId = int64(11111)
	FixAvsEvaluationExternalId = int64(22222)
)

func TestInternalEvalUpdater_AddTagsToEval(t *testing.T) {
	t.Run("should add Tags to Evaluation when enabled", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		operation := internal.Operation{
			ID:                     operationID,
			InstanceID:             instanceID,
			UpdatedAt:              time.Now(),
			State:                  domain.InProgress,
			ProvisioningParameters: FixProvisioningParameters(broker.AzurePlanID, "westeurope"),
			InstanceDetails: internal.InstanceDetails{Avs: internal.AvsLifecycleData{
				AvsEvaluationInternalId:      FixAvsEvaluationInternalId,
				AVSEvaluationExternalId:      FixAvsEvaluationExternalId,
				AVSInternalEvaluationDeleted: false,
				AVSExternalEvaluationDeleted: false,
			}},
			InputCreator: newInputCreator(),
		}

		err := memoryStorage.Operations().InsertOperation(operation)
		assert.NoError(t, err)

		mockOauthServer := newMockAvsOauthServer()
		defer mockOauthServer.Close()
		mockAvsSvc := newMockAvsService(t, true)
		mockAvsSvc.startServer()
		mockAvsSvc.evals[FixAvsEvaluationInternalId] = &avs.BasicEvaluationCreateResponse{
			Name: "test-eval",
			Tags: []*avs.Tag{
				{
					Content:      "already-exist-tag",
					TagClassId:   00,
					TagClassName: "already-exist-class-name",
				}},
			Id: FixAvsEvaluationInternalId,
		}
		defer mockAvsSvc.server.Close()

		avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
		avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
		assert.NoError(t, err)
		avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
		internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
		evalUpdater := NewInternalEvalUpdater(avsDel, internalEvalAssistant, avsConfig)

		// when
		_, repeat, err := evalUpdater.AddTagsToEval([]*avs.Tag{
			{
				Content:      "first-tag",
				TagClassId:   11,
				TagClassName: "first-tag-class-name",
			},
			{
				Content:      "second-tag",
				TagClassId:   22,
				TagClassName: "second-tag-class-name",
			},
		}, operation, "", log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, 3, len(mockAvsSvc.evals[FixAvsEvaluationInternalId].Tags))
	})

	t.Run("should skip adding Tags to Evaluation when disabled", func(t *testing.T) {
		// given
		log := logrus.New()
		memoryStorage := storage.NewMemoryStorage()
		operation := internal.Operation{
			ID:                     operationID,
			InstanceID:             instanceID,
			UpdatedAt:              time.Now(),
			State:                  domain.InProgress,
			ProvisioningParameters: FixProvisioningParameters(broker.AzurePlanID, "westeurope"),
			InstanceDetails: internal.InstanceDetails{Avs: internal.AvsLifecycleData{
				AvsEvaluationInternalId:      FixAvsEvaluationInternalId,
				AVSEvaluationExternalId:      FixAvsEvaluationExternalId,
				AVSInternalEvaluationDeleted: false,
				AVSExternalEvaluationDeleted: false,
			}},
			InputCreator: newInputCreator(),
		}

		err := memoryStorage.Operations().InsertOperation(operation)
		assert.NoError(t, err)

		mockOauthServer := newMockAvsOauthServer()
		defer mockOauthServer.Close()
		mockAvsSvc := newMockAvsService(t, true)
		mockAvsSvc.startServer()
		mockAvsSvc.evals[FixAvsEvaluationInternalId] = &avs.BasicEvaluationCreateResponse{
			Name: "test-eval",
			Tags: []*avs.Tag{
				{
					Content:      "already-exist-tag",
					TagClassId:   00,
					TagClassName: "already-exist-class-name",
				}},
			Id: FixAvsEvaluationInternalId,
		}
		defer mockAvsSvc.server.Close()

		avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
		avsConfig.AdditionalTagsEnabled = false
		avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
		assert.NoError(t, err)
		avsDel := avs.NewDelegator(avsClient, avsConfig, memoryStorage.Operations())
		internalEvalAssistant := avs.NewInternalEvalAssistant(avsConfig)
		evalUpdater := NewInternalEvalUpdater(avsDel, internalEvalAssistant, avsConfig)

		// when
		_, repeat, err := evalUpdater.AddTagsToEval([]*avs.Tag{}, operation, "", log)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), repeat)
		assert.Equal(t, 1, len(mockAvsSvc.evals[FixAvsEvaluationInternalId].Tags))
	})
}
