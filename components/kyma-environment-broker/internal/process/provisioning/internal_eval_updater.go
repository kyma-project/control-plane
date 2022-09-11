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

func (ieu *InternalEvalUpdater) AddTagsToEval(tags []*avs.Tag, operation internal.Operation, url string, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	if !ieu.avsConfig.AdditionalTagsEnabled {
		logger.Infof("Adding additional tags to AVS evaluation is disabled")
		return operation, 0 * time.Second, nil
	}

	return ieu.delegator.AddTags(logger, operation, ieu.assistant, tags)
}
