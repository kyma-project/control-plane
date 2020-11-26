package avs

import (
	"strconv"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

type Delegator struct {
	operationManager  *process.ProvisionOperationManager
	avsConfig         Config
	client            *Client
	operationsStorage storage.Operations
	configForModel    *configForModel
}

type configForModel struct {
	groupId  int64
	parentId int64
}

type avsNonSuccessResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func NewDelegator(client *Client, avsConfig Config, operationsStorage storage.Operations) *Delegator {
	return &Delegator{
		operationManager:  process.NewProvisionOperationManager(operationsStorage),
		avsConfig:         avsConfig,
		client:            client,
		operationsStorage: operationsStorage,
		configForModel: &configForModel{
			groupId:  avsConfig.GroupId,
			parentId: avsConfig.ParentId,
		},
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
		evaluationObject, err := evalAssistant.CreateBasicEvaluationRequest(operation, del.configForModel, url)
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
			return del.operationManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
		default:
			errMsg := "cannot create AVS evaluation"
			logger.Errorf("%s: %s", errMsg, err)
			return del.operationManager.OperationFailed(operation, errMsg)
		}

		evalAssistant.SetEvalId(&operation.Avs, evalResp.Id)

		updatedOperation, d = del.operationManager.UpdateOperation(operation)
	}

	evalAssistant.AppendOverrides(updatedOperation.InputCreator, updatedOperation.Avs.AvsEvaluationInternalId)

	return updatedOperation, d, nil
}

func (del *Delegator) GetEvaluation(logger logrus.FieldLogger, operation internal.ProvisioningOperation, evalAssistant EvalAssistant) (internal.ProvisioningOperation, *BasicEvaluationCreateResponse, time.Duration, error) {
	logger.Infof("starting the step avs internal id [%d] and avs external id [%d]", operation.Avs.AvsEvaluationInternalId, operation.Avs.AVSEvaluationExternalId)

	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	logger.Infof("making avs calls to get the Evaluation")
	evalId := evalAssistant.GetEvaluationId(operation.Avs)
	evalResp, err := del.client.GetEvaluation(strconv.FormatInt(evalId, 10))
	switch {
	case err == nil:
	case kebError.IsTemporaryError(err):
		errMsg := "cannot get AVS evaluation (temporary)"
		logger.Errorf("%s: %s", errMsg, err)
		retryConfig := evalAssistant.provideRetryConfig()
		op, duration, err := del.operationManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
		return op, &BasicEvaluationCreateResponse{}, duration, err
	default:
		errMsg := "cannot get AVS evaluation"
		logger.Errorf("%s: %s", errMsg, err)
		op, duration, err :=  del.operationManager.OperationFailed(operation, errMsg)
		return op, &BasicEvaluationCreateResponse{}, duration, err
	}

	updatedOperation, d = del.operationManager.UpdateOperation(operation)

	return updatedOperation, evalResp, d, nil
}

func (del *Delegator) UpdateEvaluation(logger logrus.FieldLogger, operation internal.ProvisioningOperation,  evaluation *BasicEvaluationCreateResponse, evalAssistant EvalAssistant, url string) (internal.ProvisioningOperation, time.Duration, error) {
	logger.Infof("starting the update avs internal id [%d] and avs external id [%d]", operation.Avs.AvsEvaluationInternalId, operation.Avs.AVSEvaluationExternalId)

	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	updatedEvaluation := &BasicEvaluationCreateRequest{
		DefinitionType:   evaluation.DefinitionType,
		Name:             evaluation.Name,
		Description:      evaluation.Description,
		Service:          evaluation.Service,
		URL:              evaluation.URL,
		CheckType:        evaluation.CheckType,
		Interval:         evaluation.Interval,
		TesterAccessId:   evaluation.TesterAccessId,
		Timeout:          evaluation.Timeout,
		ReadOnly:         evaluation.ReadOnly,
		ContentCheck:     evaluation.ContentCheck,
		ContentCheckType: evaluation.ContentCheckType,
		Threshold:        strconv.FormatInt(evaluation.Threshold, 10),
		GroupId:          evaluation.GroupId,
		Visibility:       evaluation.Visibility,
		Tags:             evaluation.Tags,
	}
	logger.Infof("making avs calls to update the Evaluation")

	updateResp, err := del.client.UpdateEvaluation(updatedEvaluation)
	switch {
	case err == nil:
	case kebError.IsTemporaryError(err):
		errMsg := "cannot update AVS evaluation (temporary)"
		logger.Errorf("%s: %s", errMsg, err)
		retryConfig := evalAssistant.provideRetryConfig()
		return del.operationManager.RetryOperation(operation, errMsg, retryConfig.retryInterval, retryConfig.maxTime, logger)
	default:
		errMsg := "cannot update AVS evaluation"
		logger.Errorf("%s: %s", errMsg, err)
		return del.operationManager.OperationFailed(operation, errMsg)
	}

	logger.Infof("Successfully updated evaluation %s", updateResp.Id)

	updatedOperation, d = del.operationManager.UpdateOperation(operation)

	return updatedOperation, d, nil
}

func (del *Delegator) DeleteAvsEvaluation(deProvisioningOperation internal.DeprovisioningOperation, logger logrus.FieldLogger, assistant EvalAssistant) (internal.DeprovisioningOperation, error) {
	if assistant.IsAlreadyDeleted(deProvisioningOperation.Avs) {
		logger.Infof("Evaluations have been deleted previously")
		return deProvisioningOperation, nil
	}

	if err := del.tryDeleting(assistant, deProvisioningOperation.Avs, logger); err != nil {
		return deProvisioningOperation, err
	}

	assistant.markDeleted(&deProvisioningOperation.Avs)

	updatedDeProvisioningOp, err := del.operationsStorage.UpdateDeprovisioningOperation(deProvisioningOperation)
	if err != nil {
		return deProvisioningOperation, err
	}
	return *updatedDeProvisioningOp, nil
}

func (del *Delegator) tryDeleting(assistant EvalAssistant, lifecycleData internal.AvsLifecycleData, logger logrus.FieldLogger) error {
	evaluationId := assistant.GetEvaluationId(lifecycleData)
	err := del.client.RemoveReferenceFromParentEval(evaluationId)
	if err != nil {
		logger.Errorf("error while deleting reference for evaluation %v", err)
		return err
	}

	err = del.client.DeleteEvaluation(evaluationId)
	if err != nil {
		logger.Errorf("error while deleting evaluation %v", err)
	}
	return err
}
