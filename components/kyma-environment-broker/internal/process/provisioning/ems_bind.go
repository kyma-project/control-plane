package provisioning

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"time"
)

type EmsBindStep struct {
	operationManager *process.ProvisionOperationManager
}

type eventingOverrides struct {
	oauthClientId 		string
	oauthClientSecret   string
	oauthTokenEndpoint  string
	publishUrl			string
	bebNamespace		string
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
	}
	respBinding, err := smCli.Bind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, nil, false)
	if err != nil {
		return s.handleError(operation, err, log, "Ems binding failed")
	}
	// append overrides
	evOverrides, err := getCredentials(respBinding.Binding, log)
	if err != nil {
		return s.handleError(operation, err, log, "get credentials failed")
	}
	// save the status:
	operation.Ems.Instance.Provisioned = true
	operation.Ems.Instance.ProvisioningTriggered = false
	operation, retry := s.operationManager.UpdateOperation(operation)
	if retry > 0 {
		log.Errorf("binding %s, unable to update operation", s.Name())
		return operation, time.Second, nil
	}
	// append overrides
	//log.Infof("Eventing overrides: %#v", evOverrides)
	operation.InputCreator.AppendOverrides(components.Eventing, getEventingOverrides(evOverrides))

	return operation, 0, nil
}

func getCredentials(binding servicemanager.Binding, log logrus.FieldLogger) (*eventingOverrides, error) {
	evOverrides := &eventingOverrides{}
	credentials := binding.Credentials
	evOverrides.bebNamespace = credentials["namespace"].(string)
	messaging, ok := credentials["messaging"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("false type for %s", "messaging")
	}
	for i, m := range messaging {
		m, ok := m.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d]", i))
		}
		p, ok := m["protocol"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> protocol", i))
		}
		if p[0] == "httprest" {
			evOverrides.publishUrl = m["uri"].(string)
			oa2, ok := m["oa2"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> oa2", i))
			}
			evOverrides.oauthClientId = oa2["clientid"].(string)
			evOverrides.oauthClientSecret = oa2["clientsecret"].(string)
			evOverrides.oauthTokenEndpoint = oa2["tokenendpoint"].(string)
			break
		}
	}
	return evOverrides, nil
}

func getEventingOverrides(evOverrides *eventingOverrides) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:   "authentication.oauthClientId",
			Value: evOverrides.oauthClientId,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "authentication.oauthClientSecret",
			Value: evOverrides.oauthClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "authentication.oauthTokenEndpoint",
			Value: evOverrides.oauthTokenEndpoint,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "authentication.publishUrl",
			Value: evOverrides.publishUrl,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "authentication.bebNamespace",
			Value: evOverrides.bebNamespace,
			Secret: ptr.Bool(true),
		},
	}
}

func (s *EmsBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}

