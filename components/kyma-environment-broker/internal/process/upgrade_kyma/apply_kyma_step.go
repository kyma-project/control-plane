package upgrade_kyma

import (
	"time"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
)

// NOTE: adapter for upgrade_kyma which is currently not using shared staged_manager
type ApplyKymaStep struct {
	*provisioning.ApplyKymaStep
}

func NewApplyKymaStep(os storage.Operations, cli client.Client) *ApplyKymaStep {
	return &ApplyKymaStep{provisioning.NewApplyKymaStep(os, cli)}
}

func (s *ApplyKymaStep) Run(o internal.UpgradeKymaOperation, logger logrus.FieldLogger) (internal.UpgradeKymaOperation, time.Duration, error) {
	o2, w, err := s.ApplyKymaStep.Run(o.Operation, logger)
	return internal.UpgradeKymaOperation{o2}, w, err
}
