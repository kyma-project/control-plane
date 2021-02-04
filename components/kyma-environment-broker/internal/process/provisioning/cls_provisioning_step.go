package provisioning

import (
	"fmt"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
)

type ClsInstanceProvider interface {
	ProvideClsInstanceID(om *process.ProvisionOperationManager, smCli servicemanager.Client, op internal.ProvisioningOperation, globalAccountID string, region string) (internal.ProvisioningOperation, error)
}

type clsProvisioningStep struct {
	config           *cls.Config
	instanceProvider ClsInstanceProvider
	operationManager *process.ProvisionOperationManager
}

func NewClsProvisioningStep(config *cls.Config, ip ClsInstanceProvider, repo storage.Operations) *provideClsInstanceStep {
	return &clsProvisioningStep{
		config:           config,
		operationManager: process.NewProvisionOperationManager(repo),
		instanceProvider: ip,
	}
}

func (s *clsProvisioningStep) Name() string {
	return "CLS_Provision"
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

func (s *clsProvisioningStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {

	// TODO: Fetch if there is already a CLS assigned to the GA. If so then we dont need to provision a new one.
	if operation.Cls.Instance.InstanceID != "" {
		return operation, 0, nil
	}

	smCli, err := cls.ServiceManagerClient(s.config.ServiceManager, &operation)

	op, err := s.instanceProvider.ProvideClsInstanceID(s.operationManager, smCli, operation, operation.ProvisioningParameters.ErsContext.GlobalAccountID, region)
	if err != nil {
		return s.handleError(
			operation,
			err,
			log,
			fmt.Sprintf("Unable to get tenant for GlobalaccountID/region %s/%s", operation.ProvisioningParameters.ErsContext.GlobalAccountID, region))
	}

	//operation.Cls.Instance.InstanceID = clsInstanceID
	//if operation.Cls..IsZero() {
	//	operation.Lms.RequestedAt = time.Now()
	//}

	//op, repeat := s.operationManager.UpdateOperation(operation)
	//if repeat != 0 {
	//	s.handleError(op, err, log, fmt.Sprintf("cannot save LMS tenant ID"))
	//	return operation, time.Second, nil
	//}

	return op, 0, nil
}

//func (s *provideClsInstnaceStep) provision(smCli servicemanager.Client, operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
//	// Check if we already have a cls instance assigned to the GA, if so use it
//
//	// No cls instance assigned to GA provision a new one.
//
//	var input servicemanager.ProvisioningInput
//	input.ID = uuid.New().String()
//	input.ServiceID = operation.Cls.Instance.ServiceID
//	input.PlanID = operation.Cls.Instance.PlanID
//	input.SpaceGUID = uuid.New().String()
//	input.OrganizationGUID = uuid.New().String()
//	input.Context = map[string]interface{}{
//		"platform": "kubernetes",
//	}
//	input.Parameters = clsParameters{
//		RetentionPeriod:    7,
//		MaxDataInstances:   2,
//		MaxIngestInstances: 2,
//		EsAPIEnabled:       false,
//	}
//
//	resp, err := smCli.Provision(operation.Cls.Instance.BrokerID, input, true)
//	if err != nil {
//		return s.handleError(operation, err, log, fmt.Sprintf("Provision() call failed for brokerID: %s; input: %#v", operation.Cls.Instance.BrokerID, input))
//	}
//	log.Infof("response from CLS provisioning call: %#v", resp)
//
//	operation.Cls.Instance.InstanceID = input.ID
//
//	return operation, 0, nil
//}

func (s *clsProvisioningStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}
