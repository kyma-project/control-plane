package provisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	ConnOfferingName = "connectivity"
	ConnPlanName     = "connectivity_proxy"
)

type ConnProvisionStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewConnProvisionStep(os storage.Operations) *ConnProvisionStep {
	return &ConnProvisionStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*ConnProvisionStep)(nil)

func (s *ConnProvisionStep) Name() string {
	return "CONN_Provision"
}

func (s *ConnProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (
	internal.ProvisioningOperation, time.Duration, error) {

	provisioner := NewSimpleProvisioning("Conn", &operation.Conn.Instance, s.operationManager, PassThrough)
	return provisioner.Run(operation, log)
}

func GetConnProvisioningData(info internal.ServiceManagerInstanceInfo) *servicemanager.ProvisioningInput {
	return GetSimpleInput(&info)
}
