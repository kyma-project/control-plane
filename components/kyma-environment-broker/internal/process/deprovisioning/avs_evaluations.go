package deprovisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type AvsEvaluationRemovalStep struct {
	delegator             *avs.Delegator
	operationsStorage     storage.Operations
	externalEvalAssistant avs.EvalAssistant
	internalEvalAssistant avs.EvalAssistant
	deProvisioningManager *process.OperationManager
}

func NewAvsEvaluationsRemovalStep(delegator *avs.Delegator, operationsStorage storage.Operations, externalEvalAssistant, internalEvalAssistant avs.EvalAssistant) *AvsEvaluationRemovalStep {
	return &AvsEvaluationRemovalStep{
		delegator:             delegator,
		operationsStorage:     operationsStorage,
		externalEvalAssistant: externalEvalAssistant,
		internalEvalAssistant: internalEvalAssistant,
		deProvisioningManager: process.NewOperationManager(operationsStorage),
	}
}

func (ars *AvsEvaluationRemovalStep) Name() string {
	return "De-provision_AVS_Evaluations"
}

func (ars *AvsEvaluationRemovalStep) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	logger.Infof("Avs lifecycle %+v", operation.Avs)
	if operation.Avs.AVSExternalEvaluationDeleted && operation.Avs.AVSInternalEvaluationDeleted {
		logger.Infof("Both internal and external evaluations have been deleted")
		return operation, 0, nil
	}

	operation, err := ars.delegator.DeleteAvsEvaluation(operation, logger, ars.internalEvalAssistant)
	if err != nil {
		return ars.deProvisioningManager.RetryOperation(operation, "error while deleting avs internal evaluation", err, 10*time.Second, 10*time.Minute, logger)
	}

	if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) || broker.IsFreemiumPlan(operation.ProvisioningParameters.PlanID) {
		logger.Debug("skipping AVS external evaluation deletion for trial/freemium plan")
		return operation, 0, nil
	}
	operation, err = ars.delegator.DeleteAvsEvaluation(operation, logger, ars.externalEvalAssistant)
	if err != nil {
		return ars.deProvisioningManager.RetryOperation(operation, "error while deleting avs external evaluation", err, 10*time.Second, 10*time.Minute, logger)
	}
	return operation, 0, nil

}
