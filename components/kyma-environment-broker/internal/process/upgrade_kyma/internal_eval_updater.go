package upgrade_kyma

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

func (ieu *InternalEvalUpdater) SetStatusToEval(status string, operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	return ieu.delegator.SetStatus(logger, operation, ieu.assistant, status)
}
