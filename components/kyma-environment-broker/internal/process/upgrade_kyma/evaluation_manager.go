package upgrade_kyma

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/sirupsen/logrus"
)

type EvaluationManager struct {
	avsConfig         avs.Config
	delegator         *avs.Delegator
	internalAssistant *avs.InternalEvalAssistant
	externalAssistant *avs.ExternalEvalAssistant
}

func NewEvaluationManager(delegator *avs.Delegator, config avs.Config) *EvaluationManager {
	return &EvaluationManager{
		delegator:         delegator,
		avsConfig:         config,
		internalAssistant: avs.NewInternalEvalAssistant(config),
		externalAssistant: avs.NewExternalEvalAssistant(config),
	}
}

// SetStatus updates evaluation monitors (internal and external) status.
// Note that this operation should be called twice (reason behind the zero delay)
// to configure both monitors.
func (em *EvaluationManager) SetStatus(status string, operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	avsCfg := operation.Avs

	// do internal monitor status update
	if em.internalAssistant.IsInMaintenance(avsCfg) {
		return em.delegator.SetStatus(logger, operation, em.internalAssistant, status)
	}

	// do external monitor status update
	if em.externalAssistant.IsInMaintenance(avsCfg) {
		return em.delegator.SetStatus(logger, operation, em.externalAssistant, status)
	}

	return operation, 0, nil
}

// RestoreStatus reverts previously set evaluation monitors status.
// Similarly to SetStatus, this method should also be called twice.
func (em *EvaluationManager) RestoreStatus(operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	avsCfg := operation.Avs

	// do internal monitor status reset
	if em.internalAssistant.IsInMaintenance(avsCfg) {
		return em.delegator.ResetStatus(logger, operation, em.internalAssistant)
	}

	// do external monitor status reset
	if em.externalAssistant.IsInMaintenance(avsCfg) {
		return em.delegator.ResetStatus(logger, operation, em.externalAssistant)
	}

	return operation, 0, nil
}

func (em *EvaluationManager) SetMaintenanceStatus(operation internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	return em.SetStatus(avs.StatusMaintenance, operation, logger)
}

func (em *EvaluationManager) InMaintenance(operation internal.UpgradeKymaOperation) bool {
	return em.internalAssistant.IsInMaintenance(operation.Avs) && em.externalAssistant.IsInMaintenance(operation.Avs)
}
