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
}

func NewInternalEvalUpdater(delegator *avs.Delegator, assistant *avs.InternalEvalAssistant) *InternalEvalUpdater {
	return &InternalEvalUpdater{
		delegator: delegator,
		assistant: assistant,
	}
}

func (ieu *InternalEvalUpdater) addEvalTags(operation internal.ProvisioningOperation, url string, logger logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// get current Evaluation
	op, eval, duration, err := ieu.delegator.GetEvaluation(logger, operation, ieu.assistant)
	if err != nil {
		return op, duration, err
	}

	eval.Tags = append(eval.Tags, &avs.Tag{
		Content:      "test-content-region",
		TagClassId:   61251099,
		TagClassName: "region",
	})
	
	return ieu.delegator.UpdateEvaluation(logger, operation, eval, ieu.assistant, url)
}