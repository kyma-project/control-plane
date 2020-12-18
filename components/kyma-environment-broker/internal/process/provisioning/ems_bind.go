package provisioning

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
)

type EventingOverrides struct {
	OauthClientId      string `json:"oauthClientId"`
	OauthClientSecret  string `json:"oauthClientSecret"`
	OauthTokenEndpoint string `json:"oauthTokenEndpoint"`
	PublishUrl         string `json:"publishUrl"`
	BebNamespace       string `json:"bebNamespace"`
}

type EmsBindStep struct {
	operationManager *process.ProvisionOperationManager
	secretKey        string
}

func NewEmsBindStep(os storage.Operations, secretKey string) *EmsBindStep {
	return &EmsBindStep{
		operationManager: process.NewProvisionOperationManager(os),
		secretKey:        secretKey,
	}
}

var _ Step = (*EmsBindStep)(nil)

func (s *EmsBindStep) Name() string {
	return "EMS_Bind"
}

func (s *EmsBindStep) Run(operation internal.ProvisioningOperation, log logrus.FieldLogger) (internal.ProvisioningOperation, time.Duration, error) {
	if !operation.Ems.Instance.ProvisioningTriggered {
		return s.handleError(operation, fmt.Errorf("Step %s : Ems Provisioning step was not triggered", s.Name()), log, "")
	}

	smCli, err := operation.ServiceManagerClient(log)
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Step %s : unable to create Service Manage client", s.Name()))
	}
	// test if the provisioning is finished, if not, retry after 10s
	resp, err := smCli.LastInstanceOperation(operation.Ems.Instance.InstanceKey(), "")
	if err != nil {
		return s.handleError(operation, err, log, fmt.Sprintf("Step %s : LastInstanceOperation() call failed", s.Name()))
	}
	log.Infof("Step %s : Provisioning Ems (instanceID=%s) state: %s", s.Name(), operation.Ems.Instance.InstanceID, resp.State)
	switch resp.State {
	case servicemanager.InProgress:
		return operation, 10 * time.Second, nil
	case servicemanager.Failed:
		return s.operationManager.OperationFailed(operation, fmt.Sprintf("Step %s : Ems provisioning failed: %s", s.Name(), resp.Description))
	}
	// execute binding
	var eventingOverrides *EventingOverrides
	if !operation.Ems.Instance.Provisioned {
		if operation.Ems.BindingID == "" {
			operation.Ems.BindingID = uuid.New().String()
		}
		respBinding, err := smCli.Bind(operation.Ems.Instance.InstanceKey(), operation.Ems.BindingID, nil, false)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Step %s : Bind() call failed", s.Name()))
		}
		// get overrides
		eventingOverrides, err = getCredentials(respBinding.Binding)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Step %s : getCredentials() call failed", s.Name()))
		}
		encryptedOverrides, err := encryptOverrides(s.secretKey, eventingOverrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Step %s : encryptOverrides() call failed", s.Name()))
		}
		operation.Ems.Overrides = encryptedOverrides
		operation.Ems.Instance.Provisioned = true
		operation.Ems.Instance.ProvisioningTriggered = false
		// save the status
		operation, retry := s.operationManager.UpdateOperation(operation)
		if retry > 0 {
			log.Errorf("step %s : unable to update operation", s.Name())
			return operation, time.Second, nil
		}
	} else {
		// get the credentials from encrypted string in operation.Ems.Instance.
		eventingOverrides, err = decryptOverrides(s.secretKey, operation.Ems.Overrides)
		if err != nil {
			return s.handleError(operation, err, log, fmt.Sprintf("Step %s : decryptOverrides() call failed", s.Name()))
		}
	}

	// append overrides
	operation.InputCreator.AppendOverrides(components.Eventing, getEventingOverrides(eventingOverrides))

	return operation, 0, nil
}

func (s *EmsBindStep) handleError(operation internal.ProvisioningOperation, err error, log logrus.FieldLogger, msg string) (internal.ProvisioningOperation, time.Duration, error) {
	log.Errorf("%s: %s", msg, err)
	return s.operationManager.OperationFailed(operation, msg)
}

func getCredentials(binding servicemanager.Binding) (*EventingOverrides, error) {
	evOverrides := EventingOverrides{}
	credentials := binding.Credentials
	evOverrides.BebNamespace = credentials["namespace"].(string)
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
			evOverrides.PublishUrl = m["uri"].(string)
			oa2, ok := m["oa2"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("false type for %s", fmt.Sprintf("messaging[%d] -> oa2", i))
			}
			evOverrides.OauthClientId = oa2["clientid"].(string)
			evOverrides.OauthClientSecret = oa2["clientsecret"].(string)
			evOverrides.OauthTokenEndpoint = oa2["tokenendpoint"].(string)
			break
		}
	}
	return &evOverrides, nil
}

func getEventingOverrides(evOverrides *EventingOverrides) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:    "authentication.oauthClientId",
			Value:  evOverrides.OauthClientId,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.oauthClientSecret",
			Value:  evOverrides.OauthClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.oauthTokenEndpoint",
			Value:  evOverrides.OauthTokenEndpoint,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.publishUrl",
			Value:  evOverrides.PublishUrl,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "authentication.bebNamespace",
			Value:  evOverrides.BebNamespace,
			Secret: ptr.Bool(true),
		},
	}
}

func encryptOverrides(secretKey string, overrides *EventingOverrides) (string, error) {
	ovrs, err := json.Marshal(*overrides)
	if err != nil {
		return "", errors.Wrap(err, "while encoding eventing overrides")
	}
	encrypter := storage.NewEncrypter(secretKey)
	encryptedOverrides, err := encrypter.Encrypt(ovrs)
	if err != nil {
		return "", errors.Wrap(err, "while encrypting eventing overrides")
	}
	return string(encryptedOverrides), nil
}

func decryptOverrides(secretKey string, encryptedOverrides string) (*EventingOverrides, error) {
	encrypter := storage.NewEncrypter(secretKey)
	decryptedOverrides, err := encrypter.Decrypt([]byte(encryptedOverrides))
	if err != nil {
		return nil, errors.Wrap(err, "while decrypting eventing overrides")
	}
	eventingOverrides := EventingOverrides{}
	if err := json.Unmarshal(decryptedOverrides, &eventingOverrides); err != nil {
		return nil, errors.Wrap(err, "while unmarshall eventing overrides")
	}
	return &eventingOverrides, nil
}
