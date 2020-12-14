package provisioning

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"time"
)

type EmsBindStep struct {
	operationManager *process.ProvisionOperationManager
}

func NewEmsBindStep(os storage.Operations) *EmsBindStep {
	return &EmsBindStep{
		operationManager: process.NewProvisionOperationManager(os),
	}
}

var _ Step = (*EmsBindStep)(nil)

func (s *EmsBindStep) Name() string {
	return "EMS_Bind"
}

func (s *EmsBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if operation.Ems.Instance.InstanceID == "" {
		log.Warnf("Ems Provisioning step was not triggered")
		return operation, 0, nil
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, "unable to create Service Manage client")
	}

	// test if thw provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Ems.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, "unable to create Service Manage client")
	}
	log.Infof("Provisioning Ems (instanceID=%s) state: %s", operation.Ems.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Ems provisioning failed: %s", resp.Description))
	}

	// execute binding
	if operation.Ems.BindingID == "" {
		operation.Ems.BindingID = uuid.New().String()
		operation, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("unable to update operation")
			return operation, time.Second, nil
		}
	}

	respBinding, err := smCli.Bind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, nil, false)
	if err != nil {
		return s.handleError(operation, err, log, "Ems binding failed")
	}
	log.Printf(">>> binding resp: %#v\n", resp) //TODO: delete it

	// TODO get values for EV2 from the response and put them as overrides
	err = getCredentials(respBinding.Binding, log)
	if err != nil {
		return s.handleError(operation, err, log, "get credentials failed")
	}
	// TODO save EV2 credentials in DB ?? Use "encrypt" for credentials. Only if they are different
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("unable to update operation")
		return operation, time.Second, nil
	}

	return operation, 0, nil
}

func getCredentials(binding servicemanager.Binding, log logrus.FieldLogger) error {
	// Get EV2 credentials from bindingOp
	credentials := binding.Credentials

	namespace := credentials["namespace"]
	log.Printf(">>> EV2 namespace: %s\n", namespace)
	xsappname := credentials["xsappname"]
	log.Printf(">>> xsappname: %s\n", xsappname)

	messaging, ok := credentials["messaging"].([]interface{})
	if !ok {
		return fmt.Errorf("false type for %s", "messaging")
	}
	for i, m := range messaging {
		m, ok := m.(map[string]interface{})
		if !ok {
			return fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d]", i))
		}
		p, ok := m["protocol"].([]interface{})
		if !ok {
			return fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> protocol", i))
		}
		if p[0] == "httprest" {
			uri := m["uri"]
			log.Printf(">>> EV2 uri: %s\n", uri)
			oa2, ok := m["oa2"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> oa2", i))
			}
			clientid := oa2["clientid"]
			log.Printf(">>> EV2 clientid: %s\n", clientid)
			clientsecret := oa2["clientsecret"]
			log.Printf(">>> EV2 clientsecret: %s\n", clientsecret)
			granttype := oa2["granttype"]
			log.Printf(">>> EV2 granttype: %s\n", granttype)
			tokenendpoint := oa2["tokenendpoint"]
			log.Printf(">>> EV2 tokenendpoint: %s\n", tokenendpoint)
			break
		}
	}
	return nil
}

func (s *EmsBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}

