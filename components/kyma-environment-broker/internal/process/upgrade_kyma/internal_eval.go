package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/sirupsen/logrus"
)

type InternalEvaluationStep struct {
	iec *avs.InternalEvalAssistant
}

func NewInternalEvaluationStep(assistant *avs.InternalEvalAssistant) *InternalEvaluationStep {
	return &InternalEvaluationStep{
		iec: assistant,
	}
}

func (ies *InternalEvaluationStep) Name() string {
	return "AVS_Create_Internal_Eval_Step"
}

func (ies *InternalEvaluationStep) Run(operation internal.UpgradeKymaOperation, _ logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	ies.iec.AppendOverrides(operation.InputCreator, operation.Avs.AvsEvaluationInternalId, operation.ProvisioningParameters)
	return operation, 0, nil
}
