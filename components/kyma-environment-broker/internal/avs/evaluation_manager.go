package avs

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/sirupsen/logrus"
)

type EvaluationManager struct {
	avsConfig         Config
	delegator         *Delegator
	internalAssistant *InternalEvalAssistant
	externalAssistant *ExternalEvalAssistant
}

func NewEvaluationManager(delegator *Delegator, config Config) *EvaluationManager {
	return &EvaluationManager{
		delegator:         delegator,
		avsConfig:         config,
		internalAssistant: NewInternalEvalAssistant(config),
		externalAssistant: NewExternalEvalAssistant(config),
	}
}

// SetStatus updates evaluation monitors (internal and external) status.
func (em *EvaluationManager) SetStatus(status string, avsData *internal.AvsLifecycleData, logger logrus.FieldLogger) error {
	// do internal monitor status update
	err := em.delegator.SetStatus(logger, avsData, em.internalAssistant, status)
	if err != nil {
		return err
	}

	// do external monitor status update
	err = em.delegator.SetStatus(logger, avsData, em.externalAssistant, status)
	if err != nil {
		return err
	}

	return nil
}

// RestoreStatus reverts previously set evaluation monitors status.
// On error, parent method should fail the operation progress.
// On delay, parent method should retry.
func (em *EvaluationManager) RestoreStatus(avsData *internal.AvsLifecycleData, logger logrus.FieldLogger) error {
	// do internal monitor status reset
	err := em.delegator.ResetStatus(logger, avsData, em.internalAssistant)
	if err != nil {
		return err
	}

	// do external monitor status reset
	err = em.delegator.ResetStatus(logger, avsData, em.externalAssistant)
	if err != nil {
		return err
	}

	return nil
}

func (em *EvaluationManager) SetMaintenanceStatus(avsData *internal.AvsLifecycleData, logger logrus.FieldLogger) error {
	return em.SetStatus(StatusMaintenance, avsData, logger)
}

func (em *EvaluationManager) InMaintenance(avsData internal.AvsLifecycleData) bool {
	inMaintenance := true

	// check for internal monitor
	if em.internalAssistant.IsValid(avsData) {
		inMaintenance = inMaintenance && em.internalAssistant.IsInMaintenance(avsData)
	}

	// check for external monitor
	if em.externalAssistant.IsValid(avsData) {
		inMaintenance = inMaintenance && em.externalAssistant.IsInMaintenance(avsData)
	}

	return inMaintenance
}

func (em *EvaluationManager) HasMonitors(avsData internal.AvsLifecycleData) bool {
	return em.internalAssistant.IsValid(avsData) || em.externalAssistant.IsValid(avsData)
}
