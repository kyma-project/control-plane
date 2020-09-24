package provisioning

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/lms"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

type LmsActivationStep struct {
	operationManager *process.ProvisionOperationManager
	cfg              lms.Config
	step             Step
}

func NewLmsActivationStep(os storage.Operations, cfg lms.Config, step Step) *LmsActivationStep {
	return &LmsActivationStep{
		operationManager: process.NewProvisionOperationManager(os),
		cfg:              cfg,
		step:             step,
	}
}

func (s *LmsActivationStep) Name() string {
	return s.step.Name()
}

func (s *LmsActivationStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if s.cfg.EnabledForGlobalAccounts != "" && !strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "none") {
		pp, err := operation.GetProvisioningParameters()
		if err != nil {
			log.Errorf("cannot fetch provisioning parameters from operation: %s", err)
			return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters")
		}
		if strings.EqualFold(s.cfg.EnabledForGlobalAccounts, "all") || strings.Contains(s.cfg.EnabledForGlobalAccounts, pp.ErsContext.GlobalAccountID) {
			if broker.IsTrialPlan(pp.PlanID) {
				log.Infof("Skipping step %s", s.Name())
				return operation, 0, nil
			}

			return s.step.Run(operation, log)
		}
	}
	log.Infof("Skipping step %s", s.Name())
	return operation, 0, nil
}
