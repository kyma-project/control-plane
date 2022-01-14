package avs

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Delegator struct {
	provisionManager  *process.ProvisionOperationManager
	avsConfig         Config
	client            *Client
	operationsStorage storage.Operations
}

type avsNonSuccessResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func NewDelegator(client *Client, avsConfig Config, os storage.Operations) *Delegator {
	return &Delegator{
		provisionManager:  process.NewProvisionOperationManager(os),
		avsConfig:         avsConfig,
		client:            client,
		operationsStorage: os,
	}
}

func (del *Delegator) CreateEvaluation(log logrus.FieldLogger, operation internal.ProvisioningOperation, evalAssistant EvalAssistant, url string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Infof("starting the step avs internal id [%d] and avs external id [%d]", operation.Avs.AvsEvaluationInternalId, operation.Avs.AVSEvaluationExternalId)

	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	if evalAssistant.IsValid(operation.Avs) {
		log.Infof("step has already been finished previously")
		updatedOperation = operation
	} else {
		log.Infof("making avs calls to create the Evaluation")
		evaluationObject, err := evalAssistant.CreateBasicEvaluationRequest(operation, url)
		if err != nil {
			log.Errorf("step failed with error %v", err)
			return operation, 5 * time.Second, nil
		}

		evalResp, err := del.client.CreateEvaluation(evaluationObject)
		switch {
		case err == nil:
		case kebError.IsTemporaryError(err):
			errMsg := "cannot create AVS evaluation (temporary)"
			log.Errorf("%s: %s", errMsg, err)
			retryConfig := evalAssistant.provideRetryConfig()
			return del.provisionManager.RetryOperation(operation, errMsg, err, retryConfig.retryInterval, retryConfig.maxTime, log)
		default:
			errMsg := "cannot create AVS evaluation"
			log.Errorf("%s: %s", errMsg, err)
			return del.provisionManager.OperationFailed(operation, errMsg, err, log)
		}
		updatedOperation, d, _ = del.provisionManager.UpdateOperation(operation, func(operation *internal.ProvisioningOperation) {
			evalAssistant.SetEvalId(&operation.Avs, evalResp.Id)
			evalAssistant.SetDeleted(&operation.Avs, false)
		}, log)
	}

	evalAssistant.AppendOverrides(updatedOperation.InputCreator, updatedOperation.Avs.AvsEvaluationInternalId, updatedOperation.ProvisioningParameters)

	return updatedOperation, d, nil
}

func (del *Delegator) AddTags(log logrus.FieldLogger, operation internal.ProvisioningOperation, evalAssistant EvalAssistant, tags []*Tag) (internal.ProvisioningOperation, time.Duration, error) {
	log.Infof("starting the AddTag to avs internal id [%d]", operation.Avs.AvsEvaluationInternalId)
	var updatedOperation internal.ProvisioningOperation
	d := 0 * time.Second

	log.Infof("making avs calls to add tags to the Evaluation")
	evalID := evalAssistant.GetEvaluationId(operation.Avs)

	for _, tag := range tags {
		_, err := del.client.AddTag(evalID, tag)
		switch {
		case err == nil:
		case kebError.IsTemporaryError(err):
			errMsg := "cannot add tags to AVS evaluation (temporary)"
			log.Errorf("%s: %s", errMsg, err)
			retryConfig := evalAssistant.provideRetryConfig()
			op, duration, err := del.provisionManager.RetryOperation(operation, errMsg, err, retryConfig.retryInterval, retryConfig.maxTime, log)
			return op, duration, err
		default:
			errMsg := "cannot add tags to AVS evaluation"
			log.Errorf("%s: %s", errMsg, err)
			op, duration, err := del.provisionManager.OperationFailed(operation, errMsg, err, log)
			return op, duration, err
		}
	}

	updatedOperation, d = del.provisionManager.SimpleUpdateOperation(operation)

	return updatedOperation, d, nil
}

func (del *Delegator) ResetStatus(log logrus.FieldLogger, lifecycleData *internal.AvsLifecycleData, evalAssistant EvalAssistant) error {
	status := evalAssistant.GetOriginalEvalStatus(*lifecycleData)
	// For cases when operation is not loaded (properly) from DB, status fields will be rendered
	// invalid. This will lead to a failing operation on reset in the following scenario:
	//
	// Upgrade operation when loaded (for the first time, on init) was InProgress and was later switched to complete.
	// When launching post operation logic, SetStatus will be invoked with invalid value, failing the operation.
	// One of possible hotfixes is to ensure that for invalid status there is a default value (such as Active).
	if !ValidStatus(status) {
		log.Errorf("invalid status for ResetStatus: %s", status)
		status = StatusActive
	}

	return del.SetStatus(log, lifecycleData, evalAssistant, status)
}

// RefreshStatus ensures that operation AVS lifecycle data is fetched from Avs API
func (del *Delegator) RefreshStatus(log logrus.FieldLogger, lifecycleData *internal.AvsLifecycleData, evalAssistant EvalAssistant) string {
	evalID := evalAssistant.GetEvaluationId(*lifecycleData)
	currentStatus := evalAssistant.GetEvalStatus(*lifecycleData)

	// obtain status from avs
	log.Infof("making avs calls to get evaluation data")
	eval, err := del.client.GetEvaluation(evalID)
	if err != nil || eval == nil {
		log.Errorf("cannot obtain evaluation data on RefreshStatus: %s", err)
	} else {
		currentStatus = eval.Status
	}

	evalAssistant.SetEvalStatus(lifecycleData, currentStatus)

	return currentStatus
}

func (del *Delegator) SetStatus(log logrus.FieldLogger, lifecycleData *internal.AvsLifecycleData, evalAssistant EvalAssistant, status string) error {
	// skip for non-existent or deleted evaluation
	if !evalAssistant.IsValid(*lifecycleData) {
		return nil
	}

	// fail for invalid status request
	if !ValidStatus(status) {
		errMsg := fmt.Sprintf("avs SetStatus tried invalid status: %s", status)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	evalID := evalAssistant.GetEvaluationId(*lifecycleData)
	currentStatus := del.RefreshStatus(log, lifecycleData, evalAssistant)

	log.Infof("SetStatus %s to avs id [%d]", status, evalID)

	// do api call iff current and requested status are different
	if currentStatus != status {
		log.Infof("making avs calls to set status %s to the evaluation", status)
		_, err := del.client.SetStatus(evalID, status)

		switch {
		case err == nil:
		case kebError.IsTemporaryError(err):
			errMsg := "cannot set status to AVS evaluation (temporary)"
			log.Errorf("%s: %s", errMsg, err)
			return err
		default:
			errMsg := "cannot set status to AVS evaluation"
			log.Errorf("%s: %s", errMsg, err)
			return err
		}
	}
	// update operation with newly configured status
	evalAssistant.SetEvalStatus(lifecycleData, status)

	return nil
}

func (del *Delegator) DeleteAvsEvaluation(deProvisioningOperation internal.DeprovisioningOperation, logger logrus.FieldLogger, assistant EvalAssistant) (internal.DeprovisioningOperation, error) {
	if assistant.IsAlreadyDeleted(deProvisioningOperation.Avs) {
		logger.Infof("Evaluations have been deleted previously")
		return deProvisioningOperation, nil
	}

	if err := del.tryDeleting(assistant, deProvisioningOperation, logger); err != nil {
		return deProvisioningOperation, err
	}

	assistant.SetDeleted(&deProvisioningOperation.Avs, true)

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
