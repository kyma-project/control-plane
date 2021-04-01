package provisioning

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	ConnectivityOfferingName = "connectivity"
	ConnectivityPlanName     = "connectivity_proxy"
)

type ConnectivityProvisionStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewConnectivityProvisionStep(os storage.Operations) *ConnectivityProvisionStep {
	return &ConnectivityProvisionStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*ConnectivityProvisionStep)(nil)

func (s *ConnectivityProvisionStep) Name() string {
	return "CONN_Provision"
}

func (s *ConnectivityProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (
	internal.ProvisioningOperation, time.Duration, error) {

	extractorFunc := func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
		return &op.Connectivity.Instance
	}
	provisioner := NewSimpleProvisioning("Connectivity", extractorFunc, s.operationManager, PassThrough)
	return provisioner.Run(operation, log)
}

func GetConnectivityProvisioningData(info internal.ServiceManagerInstanceInfo) *servicemanager.ProvisioningInput {
	return GetSimpleInput(&info)
}
