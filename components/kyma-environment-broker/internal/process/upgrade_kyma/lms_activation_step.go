package upgrade_kyma

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
)

type LmsActivationStep struct {
	cfg  lms.Config
	step Step
}

func NewLmsActivationStep(cfg lms.Config, step Step) *LmsActivationStep {
	return &LmsActivationStep{
		cfg:  cfg,
		step: step,
	}
}

func (s *LmsActivationStep) Name() string {
	return s.step.Name()
}

func (s *LmsActivationStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	if s.cfg.EnabledForGlobalAccounts != "" && !strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "none") {
		enabledForGA := false
		ids := strings.Split(s.cfg.EnabledForGlobalAccounts, ",")
		for i := range ids {
			if strings.EqualFold(strings.TrimSpace(ids[i]), operation.ProvisioningParameters.ErsContext.GlobalAccountID) {
				enabledForGA = true
			}
		}
		if strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "all") || enabledForGA {
			if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
				log.Infof("Skipping step %s because the step is set to skip trial plans", s.Name())
				return operation, 0, nil
			}

			return s.step.Run(operation, log)
		}
	}
	log.Infof("Skipping step %s because the step is set to skip all global accounts", s.Name())
	return operation, 0, nil
}
