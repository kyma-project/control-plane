package upgrade_kyma

import (
	"errors"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
)

type RuntimeOverridesAppender interface {
	Append(input runtimeoverrides.InputAppender, planID, kymaVersion string) error
}

//go:generate mockery --name=RuntimeVersionConfiguratorForUpgrade --output=automock --outpkg=automock --case=underscore
type RuntimeVersionConfiguratorForUpgrade interface {
	ForUpgrade(op internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error)
}

type OverridesFromSecretsAndConfigStep struct {
	operationManager       *process.UpgradeKymaOperationManager
	runtimeOverrides       RuntimeOverridesAppender
	runtimeVerConfigurator RuntimeVersionConfiguratorForUpgrade
}

func NewOverridesFromSecretsAndConfigStep(os storage.Operations, runtimeOverrides RuntimeOverridesAppender,
	rvc RuntimeVersionConfiguratorForUpgrade) *OverridesFromSecretsAndConfigStep {
	return &OverridesFromSecretsAndConfigStep{
		operationManager:       process.NewUpgradeKymaOperationManager(os),
		runtimeOverrides:       runtimeOverrides,
		runtimeVerConfigurator: rvc,
	}
}

func (s *OverridesFromSecretsAndConfigStep) Name() string {
	return "Overrides_From_Secrets_And_Config_Step"
}

func (s *OverridesFromSecretsAndConfigStep) Run(operation internal.UpgradeKymaOperation, log logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	planName, exists := broker.PlanNamesMapping[operation.ProvisioningParameters.PlanID]
	if !exists {
		log.Errorf("cannot map planID '%s' to planName", operation.ProvisioningParameters.PlanID)
		return s.operationManager.OperationFailed(operation, "invalid operation provisioning parameters", errors.New(""), log)
	}

	version, err := s.getRuntimeVersion(operation)
	if err != nil {
		return s.operationManager.RetryOperation(operation, "while getting runtime version", err, 5*time.Second, 5*time.Minute, log)
	}

	if err := s.runtimeOverrides.Append(operation.InputCreator, planName, version.Version); err != nil {
		log.Errorf(err.Error())
		return s.operationManager.RetryOperation(operation, "while appending runtime overrides", err, 10*time.Second, 30*time.Minute, log)
	}

	return operation, 0, nil
}

func (s *OverridesFromSecretsAndConfigStep) getRuntimeVersion(operation internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	// for some previously stored operations the RuntimeVersion property may not be initialized
	if operation.RuntimeVersion.Version != "" {
		return &operation.RuntimeVersion, nil
	}

	// if so, we manually compute the correct version using the same algorithm as when preparing
	// the provisioning operation. The following code can be removed after all operations will use
	// new approach for setting up runtime version in operation struct
	return s.runtimeVerConfigurator.ForUpgrade(operation)
}
