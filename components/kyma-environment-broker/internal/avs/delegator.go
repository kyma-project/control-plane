package avs

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type Delegator struct {
	provisionManager   *process.ProvisionOperationManager
	deprovisionManager *process.DeprovisionOperationManager
	upgradeManager     *process.UpgradeKymaOperationManager
	avsConfig          Config
	client             *Client
	operationsStorage  storage.Operations
}

type avsNonSuccessResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func NewDelegator(client *Client, avsConfig Config, os storage.Operations) *Delegator {
	return &Delegator{
		provisionManager:   process.NewProvisionOperationManager(os),
		deprovisionManager: process.NewDeprovisionOperationManager(os),
		upgradeManager:     process.NewUpgradeKymaOperationManager(os),
		avsConfig:          avsConfig,
		client:             client,
		operationsStorage:  os,
	}
}

func (del *Delegator) CreateEvaluation(logger logrus.FieldLogger, operation internal.ProvisioningOperation, evalAssistant EvalAssistant, url string) (internal.ProvisioningOperation, time.Duration, error) {
	logger.Infof("starting the step avs internal id [%d] and avs external id [%d]", operation.Avs.AvsEvaluationInternalId, operation.Avs.AVSEvaluationExternalId)

	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	if evalAssistant.IsAlreadyCreated(operation.Avs) {
		logger.Infof("step has already been finished previously")
		updatedOperation = operation
	} else {
		logger.Infof("making avs calls to create the Evaluation")
		evaluationObject, err := evalAssistant.CreateBasicEvaluationRequest(operation, url)
		if err != nil {
			logger.Errorf("step failed with error %v", err)
			return operation, 5 * time.Second, nil
		}

		evalResp, err := del.client.CreateEvaluation(evaluationObject)
		switch {
		case err == nil:
		case kebError.IsTemporaryError(err):
			errMsg := "cannot create AVS evaluation (temporary)"
			logger.Errorf("%s: %s", errMsg, err)
			retryConfig := evalAssistant.provideRetryConfig()
			return del.provisionManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
		default:
			errMsg := "cannot create AVS evaluation"
			logger.Errorf("%s: %s", errMsg, err)
			return del.provisionManager.OperationFailed(operation, errMsg)
		}

		evalAssistant.SetEvalId(&operation.Avs, evalResp.Id)

		updatedOperation, d = del.provisionManager.UpdateOperation(operation)
	}

	evalAssistant.AppendOverrides(updatedOperation.InputCreator, updatedOperation.Avs.AvsEvaluationInternalId, updatedOperation.ProvisioningParameters)

	return updatedOperation, d, nil
}

func (del *Delegator) AddTags(logger logrus.FieldLogger, operation internal.ProvisioningOperation, evalAssistant EvalAssistant, tags []*Tag) (internal.ProvisioningOperation, time.Duration, error) {
	logger.Infof("starting the AddTag to avs internal id [%d]", operation.Avs.AvsEvaluationInternalId)
	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	logger.Infof("making avs calls to add tags to the Evaluation")
	evalId := evalAssistant.GetEvaluationId(operation.Avs)

	for _, tag := range tags {
		_, err := del.client.AddTag(evalId, tag)
		switch {
		case err == nil:
		case kebError.IsTemporaryError(err):
			errMsg := "cannot add tags to AVS evaluation (temporary)"
			logger.Errorf("%s: %s", errMsg, err)
			retryConfig := evalAssistant.provideRetryConfig()
			op, duration, err := del.provisionManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
			return op, duration, err
		default:
			errMsg := "cannot add tags to AVS evaluation"
			logger.Errorf("%s: %s", errMsg, err)
			op, duration, err := del.provisionManager.OperationFailed(operation, errMsg)
			return op, duration, err
		}
	}

	updatedOperation, d = del.provisionManager.UpdateOperation(operation)

	return updatedOperation, d, nil
}

func (del *Delegator) SetStatus(logger logrus.FieldLogger, operation internal.UpgradeKymaOperation, evalAssistant EvalAssistant, status Status) (internal.UpgradeKymaOperation, time.Duration, error) {
	logger.Infof("starting the SetStatus to avs internal id [%d]", operation.Avs.AvsEvaluationInternalId)
	var updatedOperation internal.UpgradeKymaOperation
	d := 0 * time.Second

	logger.Infof("making avs calls to set status to the Evaluation")
	evalId := evalAssistant.GetEvaluationId(operation.Avs)

	prevStatus := operation.InstanceDetails.Avs.AvsInternalEvaluationStatus
	_, err := del.client.SetStatus(evalId, status)
	switch {
	case err == nil:
	case kebError.IsTemporaryError(err):
		errMsg := "cannot set status to AVS evaluation (temporary)"
		logger.Errorf("%s: %s", errMsg, err)
		retryConfig := evalAssistant.provideRetryConfig()
		op, duration, err := del.upgradeManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
		return op, duration, err
	default:
		errMsg := "cannot set status to AVS evaluation"
		logger.Errorf("%s: %s", errMsg, err)
		op, duration, err := del.upgradeManager.OperationFailed(operation, errMsg)
		return op, duration, err
	}

	operation.InstanceDetails.Avs.AvsOriginalInternalEvaluationStatus = prevStatus
	updatedOperation, d = del.upgradeManager.UpdateOperation(operation)

	return updatedOperation, d, nil
}

func (del *Delegator) DeleteAvsEvaluation(deProvisioningOperation internal.DeprovisioningOperation, logger logrus.FieldLogger, assistant EvalAssistant) (internal.DeprovisioningOperation, error) {
	if assistant.IsAlreadyDeleted(deProvisioningOperation.Avs) {
		logger.Infof("Evaluations have been deleted previously")
		return deProvisioningOperation, nil
	}

	if err := del.tryDeleting(assistant, deProvisioningOperation, logger); err != nil {
		return deProvisioningOperation, err
	}

	assistant.markDeleted(&deProvisioningOperation.Avs)

	updatedDeProvisioningOp, err := del.operationsStorage.UpdateDeprovisioningOperation(deProvisioningOperation)
	if err != nil {
		return deProvisioningOperation, err
	}
	return *updatedDeProvisioningOp, nil
}

func (del *Delegator) tryDeleting(assistant EvalAssistant, deProvisioningOperation internal.DeprovisioningOperation, logger logrus.FieldLogger) error {
	evaluationID := assistant.GetEvaluationId(deProvisioningOperation.Avs)
	parentID := assistant.ProvideParentId(deProvisioningOperation.ProvisioningParameters)
	err := del.client.RemoveReferenceFromParentEval(parentID, evaluationID)
	if err != nil {
		logger.Errorf("error while deleting reference for evaluation %v", err)
		return err
	}

	err = del.client.DeleteEvaluation(evaluationID)
	if err != nil {
		logger.Errorf("error while deleting evaluation %v", err)
	}
	return err
}
