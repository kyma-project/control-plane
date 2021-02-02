package provisioning

import (
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type ClsActivationStep struct {
	disabled bool
	step     Step
}

func NewClsActivationStep(disabled bool, step Step) *ClsActivationStep {
	return &ClsActivationStep{
		disabled: disabled,
		step:     step,
	}
}

func (s *ClsActivationStep) Name() string {
	return s.step.Name()
}

func (s *ClsActivationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// if s.cfg.EnabledForGlobalAccounts != "" && !strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "none") {
	// 	enabledForGA := false
	// 	ids := strings.Split(s.cfg.EnabledForGlobalAccounts, ",")
	// 	for i := range ids {
	// 		if strings.EqualFold(strings.TrimSpace(ids[i]), operation.ProvisioningParameters.ErsContext.GlobalAccountID) {
	// 			enabledForGA = true
	// 		}
	// 	}
	// 	if strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "all") || enabledForGA {
	// 		if broker.IsTrialPlan(operation.ProvisioningParameters.PlanID) {
	// 			log.Infof("Skipping step %s because the step is set to skip trial plans", s.Name())
	// 			return operation, 0, nil
	// 		}

	// 		return s.step.Run(operation, log)
	// 	}
	// }
	if s.disabled {

		return operation, 0, nil
	}

	return s.step.Run(operation, log)

	// log.Infof("Skipping step %s because the step is set to skip all global accounts", s.Name())
	// return operation, 0, nil
}
