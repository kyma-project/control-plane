package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/sirupsen/logrus"
)

type InternalEvalUpdater struct {
	delegator *avs.Delegator
	assistant *avs.InternalEvalAssistant
	avsConfig avs.Config
}

func NewInternalEvalUpdater(delegator *avs.Delegator, assistant *avs.InternalEvalAssistant, config avs.Config) *InternalEvalUpdater {
	return &InternalEvalUpdater{
		delegator: delegator,
		assistant: assistant,
		avsConfig: config,
	}
}

func (ieu *InternalEvalUpdater) AddTagsToEval(tags []*avs.Tag, operation internal.ProvisioningOperation, url string, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	op, eval, duration, err := ieu.delegator.GetEvaluation(logger, operation, ieu.assistant)
	if err != nil {
		logger.Errorf("while getting Evaluations: %s", err)
		return op, duration, err
	}

	eval.Tags = append(eval.Tags, tags...)

	return ieu.delegator.UpdateEvaluation(logger, op, eval, ieu.assistant, url)
}
