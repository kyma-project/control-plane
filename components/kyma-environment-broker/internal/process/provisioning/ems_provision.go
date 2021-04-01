package provisioning

import (
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

const (
	EmsOfferingName = "enterprise-messaging"
	EmsPlanName     = "default"
)

type EmsProvisionStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewEmsProvisionStep(os storage.Operations) *EmsProvisionStep {
	return &EmsProvisionStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*EmsProvisionStep)(nil)

func (s *EmsProvisionStep) Name() string {
	return "EMS_Provision"
}

func (s *EmsProvisionStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (
	internal.ProvisioningOperation, time.Duration, error) {

	extractorFunc := func(op *internal.ProvisioningOperation) *internal.ServiceManagerInstanceInfo {
		return &op.Ems.Instance
	}

	provisioner := NewSimpleProvisioning("Ems", extractorFunc, s.operationManager, getEventingProvisioningData)
	return provisioner.Run(operation, log)
}

func GetEventingProvisioningData(info internal.ServiceManagerInstanceInfo) *servicemanager.ProvisioningInput {
	input := GetSimpleInput(&info)
	return getEventingProvisioningData(input)
}

func getEventingProvisioningData(details *servicemanager.ProvisioningInput) *servicemanager.ProvisioningInput {
	details.Parameters = map[string]interface{}{
		"options": map[string]string{
			"management":    "true",
			"messagingrest": "true",
		},
		"rules": map[string]interface{}{
			"topicRules": map[string]interface{}{
				"publishFilter": []string{
					"${namespace}/*",
				},
				"subscribeFilter": []string{
					"${namespace}/*",
				},
			},
			"queueRules": map[string]interface{}{
				"publishFilter": []string{
					"${namespace}/*",
				},
				"subscribeFilter": []string{
					"${namespace}/*",
				},
			},
		},
		"resources": map[string]interface{}{
			"units": "30",
		},
		"version":   "1.1.0",
		"emname":    uuid.New().String(),
		"namespace": "default/sap.kyma/" + uuid.New().String(),
	}

	return details
}
