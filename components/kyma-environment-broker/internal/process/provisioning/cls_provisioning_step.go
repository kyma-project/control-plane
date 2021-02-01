package provisioning

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ClsInstanceProvider interface {
	ProvideCLSInstanceID(name, region string) (string, error)
}

// provideClsTenantStep creates (if not exists) CLS tenant and provides its ID.
// The step does not breaks the provisioning flow.
type provideClsInstnaceStep struct {
	//ClsStep
	instanceProvider   ClsInstanceProvider
	operationManager *process.ProvisionOperationManager
	regionOverride   string
}

func NewProvideClsTenantStep(ip ClsInstanceProvider, repo storage.Operations, regionOverride string, isMandatory bool) *provideClsInstnaceStep {
	return &provideClsInstnaceStep{
		operationManager: process.NewProvisionOperationManager(repo),
		instanceProvider:   ip,
		regionOverride:   regionOverride,
	}
}

func (s *provideClsInstnaceStep) Name() string {
	return "Create_CLS_Tenant"
}

//type clsParameters struct {
//	RetentionPeriod    int  `json:"retentionPeriod"`
//	MaxDataInstances   int  `json:"maxDataInstances"`
//	MaxIngestInstances int  `json:"maxIngestInstances"`
//	EsAPIEnabled       bool `json:"esApiEnabled"`
//}
//
//type ClsProvisioningStep struct {
//	operationManager *process.ProvisionOperationManager
//}
//
//func NewClsProvisioningStep(os storage.Operations) *ClsProvisioningStep {
//	return &ClsProvisioningStep{
//		operationManager: process.NewProvisionOperationManager(os),
//	}
//}
//
//var _ Step = (*ClsProvisioningStep)(nil)
//
//func (s *ClsProvisioningStep) Name() string {
//	return "CLS_Provision"
//}

func (s *provideClsInstnaceStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {

	// TODO: Fetch if there is already a CLS assigned to the GA. If so then we dont need to provision a new one.
	if operation.Cls.Instance.InstanceID != "" {
		return operation, 0, nil
	}

	// TODO: Fetch Region
	region := "cls_regions"

	clsInstanceID, err := s.instanceProvider.ProvideCLSInstanceID(operation.ProvisioningParameters.ErsContext.GlobalAccountID, region)
	if err != nil {
		return s.handleError(
			operation,
			logger,
			//time.Since(operation.UpdatedAt),
			fmt.Sprintf("Unable to get tenant for GlobalaccountID/region %s/%s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, region),
			err)
	}



	if operation.Cls.Instance.ProvisioningTriggered {
		log.Infof("CLS Provisioning step was already triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("unable to create Service Manage client"))
	}

	// provision
	operation, _, err = s.provision(smCli, operation, log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("provision()  call failed"))
	}
	// save the status
	operation.Cls.Instance.ProvisioningTriggered = true
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func (s *provideClsInstnaceStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	// Check if we already have a cls instance assigned to the GA, if so use it

	// No cls instance assigned to GA provision a new one.

	var input servicemanager.ProvisioningInput
	input.ID = uuid.New().String()
	input.ServiceID = operation.Cls.Instance.ServiceID
	input.PlanID = operation.Cls.Instance.PlanID
	input.SpaceGUID = uuid.New().String()
	input.OrganizationGUID = uuid.New().String()
	input.Context = map[string]interface{}{
		"platform": "kubernetes",
	}
	input.Parameters = clsParameters{
		RetentionPeriod:    7,
		MaxDataInstances:   2,
		MaxIngestInstances: 2,
		EsAPIEnabled:       false,
	}

	resp, err := smCli.Provision(operation.Cls.Instance.BrokerID, input, true)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Cls.Instance.BrokerID, input))
	}
	log.Infof("response from CLS provisioning call: %#v", resp)

	operation.Cls.Instance.InstanceID = input.ID

	return operation, 0, nil
}

func (s *provideClsInstnaceStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
